package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/config"
	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"github.com/bruli/raspberryWaterSystem/internal/domain/weather"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/api"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	infrahttp "github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/bruli/raspberryWaterSystem/internal/infra/listener"
	"github.com/bruli/raspberryWaterSystem/internal/infra/memory"
	"github.com/bruli/raspberryWaterSystem/internal/infra/nats"
	"github.com/bruli/raspberryWaterSystem/internal/infra/ntfy"
	"github.com/bruli/raspberryWaterSystem/internal/infra/solar"
	"github.com/bruli/raspberryWaterSystem/internal/infra/tracing"
	"github.com/bruli/raspberryWaterSystem/internal/infra/worker"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	serviceName = "raspberryWaterSystem"
	timeZone    = "Europe/Madrid"
	latitude    = 41.545
	longitude   = 2.109

	sensorReadingMaxAge     = time.Hour
	pendingIrrigationMaxAge = time.Hour
	notificationsInterval   = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pe := pinsExecutor(otel.Tracer(serviceName))
	if err := pe.TurnOffAll(ctx); err != nil {
		slog.ErrorContext(ctx, "failed turning off relays at startup", slog.String("error", err.Error()))
	}

	conf, err := config.New()
	if err != nil {
		fmt.Printf("failed building config: %s", err.Error())
		return err
	}
	log := buildLog(conf.LogLevel)

	tracingProv, err := tracing.InitTracing(ctx, serviceName)
	if err != nil {
		log.ErrorContext(ctx, "Error initializing tracing", "err", err)
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err = tracingProv.Shutdown(shutdownCtx); err != nil {
			log.ErrorContext(ctx, "Error shutting down tracing", "err", err)
		}
	}()

	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := pe.Stop(stopCtx); err != nil {
			log.ErrorContext(stopCtx, "failed turning off relays at shutdown", slog.String("error", err.Error()))
		}
	}()

	tracer := otel.Tracer(serviceName)

	eventsCh := make(chan tracing.Event)
	dispatchCh := make(chan tracing.Event)
	pendingIrrigations := disk.NewPendingIrrigationsRepository(conf.PendingIrrigationsDir, time.Now, tracer)
	restored := restorePendingIrrigations(ctx, pendingIrrigations, pendingIrrigationMaxAge, time.Now(), log)
	go queueEvents(ctx, eventsCh, dispatchCh, pendingIrrigations, restored, log)

	logCHMdw := cqs.NewCommandHndErrorMiddleware(log, tracer)
	eventsCHMdw := app.NewEventMiddleware(eventsCh, tracer)
	eventsMultiCHMdw := cqs.CommandHandlerMultiMiddleware(logCHMdw, eventsCHMdw)
	logQHMdw := cqs.NewQueryHndErrorMiddleware(log, tracer)

	tr := memory.NewLastKnownTemperatureRepository(temperatureRepository(tracer, log), sensorReadingMaxAge, time.Now, tracer)
	rr := memory.NewLastKnownRainRepository(rainRepository(tracer), sensorReadingMaxAge, time.Now, tracer)
	statusRepo := disk.NewPersistentStatusRepository(memory.NewStatusRepository(tracer), conf.StatusFile, tracer)
	zr := disk.NewZoneRepository(conf.ZonesFile, tracer)
	dailyRepo := disk.NewFertilizerProgramRepository(conf.DailyProgramsFile, tracer)
	oddRepo := disk.NewFertilizerProgramRepository(conf.OddProgramsFile, tracer)
	evenRepo := disk.NewFertilizerProgramRepository(conf.EvenProgramsFile, tracer)
	weeklyRepo := disk.NewWeeklyRepository(conf.WeeklyProgramsFile, tracer)
	tempProgRepo := disk.NewTemperatureProgramRepository(conf.TemperatureProgramsFile, tracer)
	execLogRepo := disk.NewExecutionLogRepository(conf.ExecutionLogsFile, tracer)
	location, err := time.LoadLocation(timeZone)
	if err != nil {
		log.ErrorContext(ctx, "failed loading time zone", slog.String("error", err.Error()))
		return err
	}
	sunriseSunsetRepo := api.NewSunriseSunsetRepository(latitude, longitude, location, 5*time.Second)
	go superviseWorker(ctx, log, "clean yesterday light", sunriseSunsetRepo.CleanYesterday)
	lightRepo := solar.NewFallbackLightRepository(
		sunriseSunsetRepo,
		solar.NewLightRepository(latitude, longitude, location, tracer),
		tracer,
		log,
	)
	ntfyPublisher, err := ntfy.NewPublisher(conf.NtfyUser, conf.NtfyPassword, conf.NtfyURL, conf.NtfyTopic, tracer)
	if err != nil {
		log.ErrorContext(ctx, "failed building ntfy publisher", slog.String("error", err.Error()))
		return err
	}
	eventsRepo := disk.NewEventsRepository(conf.EventsDirectory, tracer)

	eventsPublisher, err := nats.NewPublisher(conf.NatsServerURL, []string{disk.ExecutionLogsEventType, disk.TerraceWeatherEventType}, tracer)
	if err != nil {
		log.ErrorContext(ctx, "failed building events publisher", slog.String("error", err.Error()))
		return err
	}
	defer eventsPublisher.Close()
	findStatusQH := app.NewFindStatus(statusRepo, tracer)

	cronJob := buildCron(location, log)
	go terraceWeatherCron(ctx, cronJob, eventsRepo, findStatusQH, log, tracer)
	go superviseWorker(ctx, log, "reading events", func(ctx context.Context) {
		readingEvents(ctx, eventsRepo, eventsPublisher, log)
	})
	notificationsOutbox := disk.NewNotificationsOutbox(filepath.Join(conf.EventsDirectory, disk.NotificationsDirectory), time.Now, tracer)
	go superviseWorker(ctx, log, "sending notifications", func(ctx context.Context) {
		sendingNotifications(ctx, notificationsOutbox, ntfyPublisher, location, log, tracer)
	})

	qhBus := cqs.NewQueryBus()
	qhBus.Subscribe(app.FindWeatherQueryName, logQHMdw(app.NewFindWeather(tr, rr, tracer)))
	qhBus.Subscribe(app.FindStatusQueryName, logQHMdw(findStatusQH))
	qhBus.Subscribe(app.FindAllProgramsQueryName, logQHMdw(app.NewFindAllPrograms(dailyRepo, oddRepo, evenRepo, weeklyRepo, tempProgRepo, tracer)))
	qhBus.Subscribe(app.FindProgramsInTimeQueryName, logQHMdw(app.NewFindProgramsInTime(dailyRepo, oddRepo, evenRepo, weeklyRepo, tempProgRepo, tracer)))
	qhBus.Subscribe(app.FindExecutionLogsQueryName, logQHMdw(app.NewFindExecutionLogs(execLogRepo, tracer)))
	qhBus.Subscribe(app.FindZonesQueryName, logQHMdw(app.NewFindZones(zr, tracer)))

	chBus := cqs.NewCommandBus()
	chBus.Subscribe(app.CreateStatusCmdName, logCHMdw(app.NewCreateStatus(statusRepo, lightRepo, tracer)))
	chBus.Subscribe(app.UpdateStatusCmdName, logCHMdw(app.NewUpdateStatus(statusRepo, lightRepo, tracer)))
	chBus.Subscribe(app.CreateZoneCmdName, logCHMdw(app.NewCreateZone(zr, tracer)))
	chBus.Subscribe(app.ExecuteZoneCmdName, eventsMultiCHMdw(app.NewExecuteZone(zr, statusRepo, tracer)))
	chBus.Subscribe(app.ExecuteFertilizerZoneCmdName, eventsMultiCHMdw(app.NewExecuteFertilizerZone(zr, statusRepo, tracer)))
	chBus.Subscribe(app.ExecuteCleanZoneCmdName, eventsMultiCHMdw(app.NewExecuteCleanZone(zr, statusRepo, tracer)))
	chBus.Subscribe(app.ExecutePinsCmdName, logCHMdw(app.NewExecutePins(pe, statusRepo, tracer)))
	chBus.Subscribe(app.SaveExecutionLogCmdName, logCHMdw(app.NewSaveExecutionLog(execLogRepo, tracer)))
	chBus.Subscribe(app.PublishMessageCmdName, logCHMdw(app.NewPublishMessage(notificationsOutbox, tracer)))
	chBus.Subscribe(app.RemoveZoneCmdName, logCHMdw(app.NewRemoveZone(zr, tracer)))
	chBus.Subscribe(app.ActivateDeactivateServerCmdName, logCHMdw(app.NewActivateDeactivateServer(statusRepo, tracer)))
	chBus.Subscribe(app.ExecuteZoneWithStatusCmdName, eventsMultiCHMdw(app.NewExecuteZoneWithStatus(zr, statusRepo, tracer)))
	chBus.Subscribe(app.UpdateZoneCommandName, logCHMdw(app.NewUpdateZone(zr, tracer)))
	chBus.Subscribe(app.CreateDailyProgramCommandName, logCHMdw(app.NewCreateDailyProgram(dailyRepo, zr, tracer)))
	chBus.Subscribe(app.CreateOddProgramCommandName, logCHMdw(app.NewCreateOddProgram(oddRepo, zr, tracer)))
	chBus.Subscribe(app.CreateEvenProgramCommandName, logCHMdw(app.NewCreateEvenProgram(evenRepo, zr, tracer)))
	chBus.Subscribe(app.RemoveDailyProgramCommandName, logCHMdw(app.NewRemoveDailyProgram(dailyRepo, tracer)))
	chBus.Subscribe(app.RemoveOddProgramCommandName, logCHMdw(app.NewRemoveOddProgram(oddRepo, tracer)))
	chBus.Subscribe(app.RemoveEvenProgramCommandName, logCHMdw(app.NewRemoveEvenProgram(evenRepo, tracer)))
	chBus.Subscribe(app.CreateWeeklyProgramCommandName, logCHMdw(app.NewCreateWeeklyProgram(weeklyRepo, tracer)))
	chBus.Subscribe(app.RemoveWeeklyProgramCommandName, logCHMdw(app.NewRemoveWeeklyProgram(weeklyRepo, tracer)))
	chBus.Subscribe(app.CreateTemperatureProgramCommandName, logCHMdw(app.NewCreateTemperatureProgram(tempProgRepo, tracer)))
	chBus.Subscribe(app.RemoveTemperatureProgramCommandName, logCHMdw(app.NewRemoveTemperatureProgram(tempProgRepo, tracer)))
	chBus.Subscribe(app.UpdateTemperatureProgramCommandName, logCHMdw(app.NewUpdateTemperatureProgram(tempProgRepo, tracer)))
	chBus.Subscribe(app.ChangeCleanStatusCmdName, logCHMdw(app.NewChangeCleanStatus(statusRepo, tracer)))

	eventBus := event.NewBus()
	eventBus.Subscribe(zone.ExecutedEventName, listener.NewExecutePinsOnExecuteZone(chBus, tracer), listener.NewWriteExecutionLogEventOnExecuteZone(eventsRepo, tracer))

	eventBus.Subscribe(zone.IgnoredEventName, listener.NewPublishMessageOnZoneIgnored(chBus, tracer))

	eventBus.Subscribe(zone.FertilizerZoneExecutedEventName, listener.NewExecutePinsOnExecuteFertilizerZone(chBus, tracer, log), listener.NewWriteExecutionLogEventOnExecuteFertilizerZone(eventsRepo, tracer))

	eventBus.Subscribe(zone.CleanZoneExecutedEventName, listener.NewExecutePinsOnExecuteCleanZone(chBus, tracer, log))

	if err = initStatus(ctx, chBus, qhBus); err != nil {
		log.ErrorContext(ctx, "failed initializing status", slog.String("error", err.Error()))
		return err
	}

	errCh := make(chan error, 2)

	go superviseWorker(ctx, log, "update status", func(ctx context.Context) {
		updateStatusWorker(ctx, qhBus, chBus, log)
	})
	go superviseWorker(ctx, log, "events", func(ctx context.Context) {
		eventsWorker(ctx, dispatchCh, eventBus, log, tracer)
	})
	go superviseWorker(ctx, log, "execution in time", func(ctx context.Context) {
		executionInTimeWorker(ctx, qhBus, chBus, location, log, tracer)
	})

	go func() {
		if err := runHTTPServer(ctx, chBus, qhBus, conf, log, tracer); err != nil {
			log.ErrorContext(ctx, "failed running http server", slog.String("error", err.Error()))
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received")
		return nil
	}
}

func terraceWeatherCron(ctx context.Context, cronJob *cron.Cron, repo *disk.EventsRepository, ch cqs.QueryHandler, log *slog.Logger, tracer trace.Tracer) {
	defer cronJob.Stop()
	_, err := cronJob.AddFunc("00 * * * *", func() {
		ctx, span := tracer.Start(ctx, "TerraceWeatherCron")
		defer span.End()
		log.InfoContext(ctx, "[WORKER] Terrace Weather: getting weather")
		result, err := ch.Handle(ctx, app.FindStatusQuery{})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.ErrorContext(ctx, "[WORKER] Terrace Weather: failed getting weather", slog.String("error", err.Error()))
			return
		}
		st, ok := result.(status.Status)
		if !ok {
			err := UnexpectedQueryResultError{result: result}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.ErrorContext(ctx, "[WORKER] Terrace Weather: failed casting weather", slog.String("error", err.Error()))
			return
		}
		tw := disk.Weather{
			Temperature: st.Weather().Temperature().Float32(),
			IsRaining:   st.Weather().IsRaining(),
			Humidity:    st.Weather().Humidity().Float32(),
		}
		ev, err := disk.NewFromWeather(ctx, &tw)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.ErrorContext(ctx, "[WORKER] Terrace Weather: failed creating weather event", slog.String("error", err.Error()))
			return
		}
		if err = repo.Save(ctx, ev); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.ErrorContext(ctx, "[WORKER] Terrace Weather: failed saving weather event", slog.String("error", err.Error()))
			return
		}
		span.SetStatus(codes.Ok, "Weather updated")
	})
	if err != nil {
		log.ErrorContext(ctx, "[WORKER] Terrace Weather: failed adding cron job", slog.String("error", err.Error()))
		return
	}
	log.InfoContext(ctx, "[WORKER] Terrace Weather: cron job started")
	cronJob.Start()
	<-ctx.Done()
	log.InfoContext(ctx, "[WORKER] Terrace Weather: context done")
}

func buildCron(location *time.Location, log *slog.Logger) *cron.Cron {
	cronLog := cron.PrintfLogger(slog.NewLogLogger(log.Handler(), slog.LevelError))
	return cron.New(cron.WithLocation(location), cron.WithChain(cron.Recover(cronLog)))
}

func sendingNotifications(
	ctx context.Context,
	pending worker.PendingNotifications,
	sender worker.NotificationSender,
	location *time.Location,
	log *slog.Logger,
	tracer trace.Tracer,
) {
	log.InfoContext(ctx, "[WORKER] Sending notifications: started")
	ticker := time.NewTicker(notificationsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.InfoContext(ctx, "[WORKER] Sending notifications: context done")
			return
		case <-ticker.C:
			if _, err := worker.SendNotifications(ctx, pending, sender, time.Now(), location, tracer); err != nil {
				log.WarnContext(ctx, "[WORKER] Sending notifications: notifications kept for the next attempt", slog.String("error", err.Error()))
			}
		}
	}
}

func readingEvents(
	ctx context.Context,
	eventsRepo *disk.EventsRepository,
	publisher *nats.Publisher,
	log *slog.Logger,
) {
	log.InfoContext(ctx, "[WORKER] Reading Events: starting")

	tick := time.NewTicker(time.Minute)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			log.InfoContext(ctx, "[WORKER] Reading Events: context done")
			return

		case <-tick.C:
			if !publisher.IsConnected() {
				log.WarnContext(ctx, "[WORKER] Reading Events: nats not connected, keeping events in outbox")
				continue
			}
			events, err := eventsRepo.FindAll(ctx)
			if err != nil {
				log.ErrorContext(ctx, "[WORKER] Reading Events: failed reading events",
					slog.String("error", err.Error()))
				continue
			}

			for _, ev := range events {
				prop := propagation.TraceContext{}
				carrier := propagation.MapCarrier{
					"traceparent": ev.Trace["traceparent"],
				}
				batchCtx := prop.Extract(ctx, carrier)
				tracer := otel.Tracer("reading-events")
				ctx, span := tracer.Start(batchCtx, ev.EventType)
				err = publishEvent(ctx, eventsRepo, publisher, &ev, span)
				span.End()
				if err == nil {
					continue
				}
				log.ErrorContext(ctx, "[WORKER] Reading Events: failed publishing event",
					slog.String("event_id", ev.ID),
					slog.String("error", err.Error()))
				if errors.As(err, &nats.NotConnectedError{}) {
					break
				}
			}
		}
	}
}

func publishEvent(ctx context.Context, evRepo *disk.EventsRepository, publisher *nats.Publisher, ev *disk.Event, span trace.Span) error {
	if err := publisher.PublishEvent(ctx, ev.ID, ev.EventType, ev.Payload); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if err := evRepo.Remove(ctx, ev); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "event published and removed")
	return nil
}

func runHTTPServer(ctx context.Context, chBus cqs.CommandBus, qhBus cqs.QueryBus, conf *config.Config, log *slog.Logger, tracer trace.Tracer) error {
	definitions := handlersDefinition(chBus, qhBus, conf.AuthToken, tracer)
	httpHandlers := infrahttp.NewHandler(definitions)
	if err := infrahttp.RunServer(ctx, conf.ServerURL, httpHandlers, &infrahttp.CORSOpt{}, log); err != nil {
		log.ErrorContext(ctx, "[HTTP SERVER] failed running server", slog.String("error", err.Error()))
		return err
	}
	return nil
}

func buildLog(level string) *slog.Logger {
	levels := map[string]slog.Level{
		"info":  slog.LevelInfo,
		"debug": slog.LevelDebug,
		"error": slog.LevelError,
		"warn":  slog.LevelWarn,
	}
	l := levels[level]

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: l,
	})

	return slog.New(handler).With(slog.String("service", serviceName))
}

func executionInTimeWorker(ctx context.Context, qh cqs.QueryHandler, ch cqs.CommandHandler, location *time.Location, logger *slog.Logger, tracer trace.Tracer) {
	logger.InfoContext(ctx, "[WORKER] Execution in time: started")
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "[WORKER] Execution in time: context done")
			return
		case <-ticker.C:
			if err := worker.ExecutionInTime(ctx, qh, ch, time.Now().In(location), tracer); err != nil {
				logger.ErrorContext(ctx, "[WORKER] failed execution in time", slog.String("error", err.Error()))
			}
		}
	}
}

func eventsWorker(ctx context.Context, ch <-chan tracing.Event, evBus event.Bus, logger *slog.Logger, tracer trace.Tracer) {
	logger.InfoContext(ctx, "[WORKER] Events: started")
	for {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "[WORKER] Events: context done")
			return
		case traced, ok := <-ch:
			if !ok {
				logger.InfoContext(ctx, "[WORKER] Events: channel closed")
				return
			}
			parentCtx := trace.ContextWithSpanContext(ctx, traced.SpanContext)
			ctx, span := tracer.Start(parentCtx, "eventDispatch")
			span.SetAttributes(attribute.String("event-name", traced.Event.EventName()))
			if err := evBus.Dispatch(ctx, traced.Event); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				logger.ErrorContext(ctx, "[WORKER] Events: failed dispatching event", slog.String("error", err.Error()))
			}
			span.End()
		}
	}
}

func updateStatusWorker(ctx context.Context, qh cqs.QueryHandler, ch cqs.CommandHandler, logger *slog.Logger) {
	logger.InfoContext(ctx, "[WORKER] Update status: started")
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "[WORKER] Update status: context done")
			return
		case <-ticker.C:
			updateStatus(ctx, qh, ch, logger)
		}
	}
}

func updateStatus(ctx context.Context, qh cqs.QueryHandler, ch cqs.CommandHandler, logger *slog.Logger) {
	result, err := qh.Handle(ctx, app.FindWeatherQuery{})
	if err != nil {
		logger.ErrorContext(ctx, "[WORKER] Update status: failed getting weather", slog.String("error", err.Error()))
	}
	weath, _ := result.(weather.Weather)
	if _, err = ch.Handle(ctx, app.UpdateStatusCmd{Weather: weath}); err != nil {
		logger.ErrorContext(ctx, "[WORKER] Update status: failed updating status", slog.String("error", err.Error()))
	}
}

func initStatus(ctx context.Context, ch cqs.CommandHandler, qh cqs.QueryHandler) error {
	result, err := qh.Handle(ctx, app.FindWeatherQuery{})
	if err != nil {
		return err
	}
	weath, _ := result.(weather.Weather)
	_, err = ch.Handle(ctx, app.CreateStatusCmd{
		StartedAt: time.Now(),
		Weather:   weath,
	})
	return err
}

func handlersDefinition(chBus cqs.CommandBus, qhBus cqs.QueryBus, authToken string, tracer trace.Tracer) infrahttp.HandlersDefinition {
	authMdw := infrahttp.AuthMiddleware(authToken)
	return infrahttp.HandlersDefinition{
		{
			Endpoint:    "/",
			Method:      http.MethodGet,
			HandlerFunc: infrahttp.Homepage(),
		},
		{
			Endpoint:    "/zones",
			Method:      http.MethodPost,
			HandlerFunc: authMdw(infrahttp.CreateZone(chBus, tracer)),
		},
		{
			Endpoint:    "/zones",
			Method:      http.MethodGet,
			HandlerFunc: authMdw(infrahttp.FindZones(qhBus, tracer)),
		},
		{
			Endpoint:    "/zones/{id}/execute",
			Method:      http.MethodPost,
			HandlerFunc: authMdw(infrahttp.ExecuteZone(chBus, tracer)),
		},
		{
			Endpoint:    "/zones/fertilizer/{id}/execute",
			Method:      http.MethodPost,
			HandlerFunc: authMdw(infrahttp.ExecuteFertilizerZone(chBus, tracer)),
		},
		{
			Endpoint:    "/zones/clean/{id}/execute",
			Method:      http.MethodPost,
			HandlerFunc: authMdw(infrahttp.ExecuteCleanZone(chBus, tracer)),
		},
		{
			Endpoint:    "/zones/{id}",
			Method:      http.MethodDelete,
			HandlerFunc: authMdw(infrahttp.RemoveZone(chBus, tracer)),
		},
		{
			Endpoint:    "/zones/{id}",
			Method:      http.MethodPut,
			HandlerFunc: authMdw(infrahttp.UpdateZone(chBus, tracer)),
		},
		{
			Endpoint:    "/status",
			Method:      http.MethodGet,
			HandlerFunc: authMdw(infrahttp.FindStatus(qhBus, tracer)),
		},
		{
			Endpoint:    "/status/{action}",
			Method:      http.MethodPatch,
			HandlerFunc: authMdw(infrahttp.ActivateDeactivateServer(chBus, tracer)),
		},
		{
			Endpoint:    "/weather",
			Method:      http.MethodGet,
			HandlerFunc: authMdw(infrahttp.FindWeather(qhBus, tracer)),
		},
		{
			Endpoint:    "/programs",
			Method:      http.MethodGet,
			HandlerFunc: authMdw(infrahttp.FindAllPrograms(qhBus, tracer)),
		},
		{
			Endpoint:    "/programs/daily",
			Method:      http.MethodPost,
			HandlerFunc: authMdw(infrahttp.CreateProgram(chBus, infrahttp.DailyProgram, tracer)),
		},
		{
			Endpoint:    "/programs/daily/{hour}",
			Method:      http.MethodDelete,
			HandlerFunc: authMdw(infrahttp.RemoveProgram(chBus, infrahttp.DailyProgram, tracer)),
		},
		{
			Endpoint:    "/programs/odd",
			Method:      http.MethodPost,
			HandlerFunc: authMdw(infrahttp.CreateProgram(chBus, infrahttp.OddProgram, tracer)),
		},
		{
			Endpoint:    "/programs/odd/{hour}",
			Method:      http.MethodDelete,
			HandlerFunc: authMdw(infrahttp.RemoveProgram(chBus, infrahttp.OddProgram, tracer)),
		},
		{
			Endpoint:    "/programs/even",
			Method:      http.MethodPost,
			HandlerFunc: authMdw(infrahttp.CreateProgram(chBus, infrahttp.EvenProgram, tracer)),
		},
		{
			Endpoint:    "/programs/even/{hour}",
			Method:      http.MethodDelete,
			HandlerFunc: authMdw(infrahttp.RemoveProgram(chBus, infrahttp.EvenProgram, tracer)),
		},
		{
			Endpoint:    "/programs/weekly",
			Method:      http.MethodPost,
			HandlerFunc: authMdw(infrahttp.CreateWeeklyProgram(chBus, tracer)),
		},
		{
			Endpoint:    "/programs/weekly/{day}",
			Method:      http.MethodDelete,
			HandlerFunc: authMdw(infrahttp.RemoveWeeklyProgram(chBus, tracer)),
		},
		{
			Endpoint:    "/programs/temperature",
			Method:      http.MethodPost,
			HandlerFunc: authMdw(infrahttp.CreateTemperatureProgram(chBus, tracer)),
		},
		{
			Endpoint:    "/programs/temperature/{temperature}",
			Method:      http.MethodPut,
			HandlerFunc: authMdw(infrahttp.UpdateTemperatureProgram(chBus, tracer)),
		},
		{
			Endpoint:    "/programs/temperature/{temperature}",
			Method:      http.MethodDelete,
			HandlerFunc: authMdw(infrahttp.RemoveTemperatureProgram(chBus, tracer)),
		},
		{
			Endpoint:    "/logs",
			Method:      http.MethodGet,
			HandlerFunc: authMdw(infrahttp.FindExecutionLogs(qhBus, tracer)),
		},
		{
			Endpoint:    "/metrics",
			Method:      http.MethodGet,
			HandlerFunc: infrahttp.Metrics(),
		},
	}
}

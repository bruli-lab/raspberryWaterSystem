package main

import (
	"context"
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
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/api"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	infrahttp "github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/bruli/raspberryWaterSystem/internal/infra/listener"
	"github.com/bruli/raspberryWaterSystem/internal/infra/memory"
	"github.com/bruli/raspberryWaterSystem/internal/infra/nats"
	"github.com/bruli/raspberryWaterSystem/internal/infra/ntfy"
	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"github.com/bruli/raspberryWaterSystem/internal/infra/solar"
	"github.com/bruli/raspberryWaterSystem/internal/infra/tracing"
	"github.com/bruli/raspberryWaterSystem/internal/infra/worker"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	statusUpdateInterval    = 5 * time.Minute
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

	eventsCh := make(chan app.TracedEvent)
	dispatchCh := make(chan app.TracedEvent)
	pendingIrrigations := disk.NewPendingIrrigationsRepository(conf.PendingIrrigationsDir, time.Now, tracer)
	restored := worker.RestorePendingIrrigations(ctx, pendingIrrigations, pendingIrrigationMaxAge, time.Now(), log)
	go worker.QueueEvents(ctx, eventsCh, dispatchCh, pendingIrrigations, restored, log)

	logCHMdw := cqs.NewCommandHndErrorMiddleware(log, tracer)
	eventsCHMdw := app.NewEventMiddleware(eventsCh, tracer)
	eventsMultiCHMdw := cqs.CommandHandlerMultiMiddleware(logCHMdw, eventsCHMdw)
	logQHMdw := cqs.NewQueryHndErrorMiddleware(log, tracer)

	tr := memory.NewLastKnownTemperatureRepository(temperatureRepository(tracer, log), sensorReadingMaxAge, time.Now, tracer)
	rr := memory.NewLastKnownRainRepository(rainRepository(tracer), sensorReadingMaxAge, time.Now, tracer)
	statusRepo := disk.NewPersistentStatusRepository(memory.NewStatusRepository(tracer), conf.StatusFile, tracer)
	zr := disk.NewZoneRepository(conf.ZonesFile, tracer)
	dailyRepo := disk.NewProgramRepository(conf.DailyProgramsFile, tracer)
	oddRepo := disk.NewProgramRepository(conf.OddProgramsFile, tracer)
	evenRepo := disk.NewProgramRepository(conf.EvenProgramsFile, tracer)
	weeklyRepo := disk.NewWeeklyRepository(conf.WeeklyProgramsFile, tracer)
	tempProgRepo := disk.NewTemperatureProgramRepository(conf.TemperatureProgramsFile, tracer)
	execLogRepo := disk.NewExecutionLogRepository(conf.ExecutionLogsFile, tracer)
	location, err := time.LoadLocation(timeZone)
	if err != nil {
		log.ErrorContext(ctx, "failed loading time zone", slog.String("error", err.Error()))
		return err
	}
	sunriseSunsetRepo := api.NewSunriseSunsetRepository(api.SunriseSunsetURL, latitude, longitude, location, 5*time.Second)
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

	eventsPublisher, err := nats.NewPublisher(conf.NatsServerURL, []string{outbox.ExecutionLogsEventType, outbox.TerraceWeatherEventType}, tracer)
	if err != nil {
		log.ErrorContext(ctx, "failed building events publisher", slog.String("error", err.Error()))
		return err
	}
	defer eventsPublisher.Close()
	findStatusQH := app.NewFindStatus(statusRepo, tracer)

	cronJob := buildCron(location, log)
	go terraceWeatherCron(ctx, cronJob, eventsRepo, findStatusQH, log, tracer)
	go superviseWorker(ctx, log, "reading events", func(ctx context.Context) {
		readingEvents(ctx, eventsRepo, eventsPublisher, log, tracer)
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
	eventBus.Subscribe(zone.ExecutedEventName, listener.NewExecutePinsOnExecuteZone(chBus, tracer), listener.NewWriteExecutionLogEventOnExecuteZone(eventsRepo, time.Now, tracer))

	eventBus.Subscribe(zone.IgnoredEventName, listener.NewPublishMessageOnZoneIgnored(chBus, tracer))

	eventBus.Subscribe(zone.FertilizerZoneExecutedEventName, listener.NewExecutePinsOnExecuteFertilizerZone(chBus, listener.WaitFor, tracer, log), listener.NewWriteExecutionLogEventOnExecuteFertilizerZone(eventsRepo, time.Now, tracer))

	eventBus.Subscribe(zone.CleanZoneExecutedEventName, listener.NewExecutePinsOnExecuteCleanZone(chBus, listener.WaitFor, tracer, log))

	if err = worker.InitStatus(ctx, qhBus, chBus, time.Now()); err != nil {
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

func terraceWeatherCron(ctx context.Context, cronJob *cron.Cron, events worker.EventsWriter, qh cqs.QueryHandler, log *slog.Logger, tracer trace.Tracer) {
	defer cronJob.Stop()
	_, err := cronJob.AddFunc("00 * * * *", func() {
		if err := worker.SaveTerraceWeather(ctx, qh, events, time.Now(), tracer); err != nil {
			log.ErrorContext(ctx, "[WORKER] Terrace Weather: failed saving weather event", slog.String("error", err.Error()))
		}
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

func runEvery(ctx context.Context, interval time.Duration, log *slog.Logger, name string, tick func(context.Context)) {
	log.InfoContext(ctx, "[WORKER] "+name+": started")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.InfoContext(ctx, "[WORKER] "+name+": context done")
			return
		case <-ticker.C:
			tick(ctx)
		}
	}
}

func sendingNotifications(
	ctx context.Context,
	pending worker.PendingNotifications,
	sender worker.NotificationSender,
	location *time.Location,
	log *slog.Logger,
	tracer trace.Tracer,
) {
	runEvery(ctx, notificationsInterval, log, "Sending notifications", func(ctx context.Context) {
		if _, err := worker.SendNotifications(ctx, pending, sender, time.Now(), location, tracer); err != nil {
			log.WarnContext(ctx, "[WORKER] Sending notifications: notifications kept for the next attempt", slog.String("error", err.Error()))
		}
	})
}

func readingEvents(ctx context.Context, pending worker.PendingEvents, publisher worker.EventPublisher, log *slog.Logger, tracer trace.Tracer) {
	runEvery(ctx, time.Minute, log, "Reading Events", func(ctx context.Context) {
		if _, err := worker.PublishOutboxEvents(ctx, pending, publisher, tracer); err != nil {
			log.WarnContext(ctx, "[WORKER] Reading Events: events kept in the outbox", slog.String("error", err.Error()))
		}
	})
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
	runEvery(ctx, time.Minute, logger, "Execution in time", func(ctx context.Context) {
		if err := worker.ExecutionInTime(ctx, qh, ch, time.Now().In(location), tracer); err != nil {
			logger.ErrorContext(ctx, "[WORKER] failed execution in time", slog.String("error", err.Error()))
		}
	})
}

func eventsWorker(ctx context.Context, ch <-chan app.TracedEvent, evBus event.Bus, logger *slog.Logger, tracer trace.Tracer) {
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
	runEvery(ctx, statusUpdateInterval, logger, "Update status", func(ctx context.Context) {
		if err := worker.UpdateStatus(ctx, qh, ch); err != nil {
			logger.ErrorContext(ctx, "[WORKER] Update status: failed updating status", slog.String("error", err.Error()))
		}
	})
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

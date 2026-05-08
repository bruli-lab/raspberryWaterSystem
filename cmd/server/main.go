package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/config"
	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"github.com/bruli/raspberryWaterSystem/internal/domain/weather"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/api"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	infrahttp "github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/bruli/raspberryWaterSystem/internal/infra/listener"
	"github.com/bruli/raspberryWaterSystem/internal/infra/memory"
	"github.com/bruli/raspberryWaterSystem/internal/infra/nats"
	"github.com/bruli/raspberryWaterSystem/internal/infra/telegram"
	"github.com/bruli/raspberryWaterSystem/internal/infra/tracing"
	"github.com/bruli/raspberryWaterSystem/internal/infra/worker"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "raspberryWaterSystem"

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	log := buildLog()
	conf, err := config.New()
	if err != nil {
		log.ErrorContext(ctx, "failed building config", slog.String("error", err.Error()))
		return err
	}

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

	tracer := otel.Tracer(serviceName)

	eventsCh := make(chan tracing.Event, 5)
	defer close(eventsCh)

	logCHMdw := cqs.NewCommandHndErrorMiddleware(log, tracer)
	eventsCHMdw := app.NewEventMiddleware(eventsCh, tracer)
	eventsMultiCHMdw := cqs.CommandHandlerMultiMiddleware(logCHMdw, eventsCHMdw)
	logQHMdw := cqs.NewQueryHndErrorMiddleware(log, tracer)

	tr := temperatureRepository(tracer)
	rr := rainRepository(tracer)
	sr := memory.NewStatusRepository(tracer)
	zr := disk.NewZoneRepository(conf.ZonesFile, tracer)
	dailyRepo := disk.NewProgramRepository(conf.DailyProgramsFile, tracer)
	oddRepo := disk.NewProgramRepository(conf.OddProgramsFile, tracer)
	evenRepo := disk.NewProgramRepository(conf.EvenProgramsFile, tracer)
	weeklyRepo := disk.NewWeeklyRepository(conf.WeeklyProgramsFile, tracer)
	tempProgRepo := disk.NewTemperatureProgramRepository(conf.TemperatureProgramsFile, tracer)
	execLogRepo := disk.NewExecutionLogRepository(conf.ExecutionLogsFile, tracer)
	lightRepo := api.NewSunriseSunsetRepository(5 * time.Second)
	go lightRepo.CleanYesterday(ctx)
	pe := pinsExecutor(tracer)
	messagePublisher := telegram.NewMessagePublisher(conf.TelegramToken, conf.TelegramChatID, tracer)
	eventsRepo := disk.NewEventsRepository(conf.EventsDirectory, tracer)

	eventsPublisher, err := nats.NewPublisher(conf.NatsServerURL, tracer)
	if err != nil {
		log.ErrorContext(ctx, "failed building execution logs", slog.String("error", err.Error()))
		return err
	}
	defer eventsPublisher.Close()
	if err := eventsPublisher.EnsureStream([]string{disk.ExecutionLogsEventType, disk.TerraceWeatherEventType}); err != nil {
		log.ErrorContext(ctx, "failed ensuring execution logs stream", slog.String("error", err.Error()))
		return err
	}
	findStatusQH := app.NewFindStatus(sr, tracer)

	cronJob, err := buildCron()
	if err != nil {
		log.ErrorContext(ctx, "failed building cron")
		return err
	}
	go terraceWeatherCron(ctx, cronJob, eventsRepo, findStatusQH, log, tracer)
	go readingEvents(ctx, eventsRepo, eventsPublisher, log)

	qhBus := app.NewQueryBus()
	qhBus.Subscribe(app.FindWeatherQueryName, logQHMdw(app.NewFindWeather(tr, rr, tracer)))
	qhBus.Subscribe(app.FindStatusQueryName, logQHMdw(findStatusQH))
	qhBus.Subscribe(app.FindAllProgramsQueryName, logQHMdw(app.NewFindAllPrograms(dailyRepo, oddRepo, evenRepo, weeklyRepo, tempProgRepo, tracer)))
	qhBus.Subscribe(app.FindProgramsInTimeQueryName, logQHMdw(app.NewFindProgramsInTime(dailyRepo, oddRepo, evenRepo, weeklyRepo, tempProgRepo, tracer)))
	qhBus.Subscribe(app.FindExecutionLogsQueryName, logQHMdw(app.NewFindExecutionLogs(execLogRepo, tracer)))
	qhBus.Subscribe(app.FindZonesQueryName, logQHMdw(app.NewFindZones(zr, tracer)))

	chBus := app.NewCommandBus()
	chBus.Subscribe(app.CreateStatusCmdName, logCHMdw(app.NewCreateStatus(sr, lightRepo, tracer)))
	chBus.Subscribe(app.UpdateStatusCmdName, logCHMdw(app.NewUpdateStatus(sr, lightRepo, tracer)))
	chBus.Subscribe(app.CreateZoneCmdName, logCHMdw(app.NewCreateZone(zr, tracer)))
	chBus.Subscribe(app.ExecuteZoneCmdName, eventsMultiCHMdw(app.NewExecuteZone(zr, tracer)))
	chBus.Subscribe(app.ExecuteFertilizerZoneCmdName, eventsMultiCHMdw(app.NewExecuteFertilizerZone(zr, tracer)))
	chBus.Subscribe(app.ExecutePinsCmdName, logCHMdw(app.NewExecutePins(pe, tracer)))
	chBus.Subscribe(app.SaveExecutionLogCmdName, logCHMdw(app.NewSaveExecutionLog(execLogRepo, tracer)))
	chBus.Subscribe(app.PublishMessageCmdName, logCHMdw(app.NewPublishMessage(messagePublisher, tracer)))
	chBus.Subscribe(app.RemoveZoneCmdName, logCHMdw(app.NewRemoveZone(zr, tracer)))
	chBus.Subscribe(app.ActivateDeactivateServerCmdName, logCHMdw(app.NewActivateDeactivateServer(sr, tracer)))
	chBus.Subscribe(app.ExecuteZoneWithStatusCmdName, eventsMultiCHMdw(app.NewExecuteZoneWithStatus(zr, sr, tracer)))
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

	eventBus := cqs.NewEventBus()
	eventBus.Subscribe(zone.Executed{
		BasicEvent: cqs.BasicEvent{NameAttr: zone.ExecutedEventName},
	}, listener.NewExecutePinsOnExecuteZone(chBus, tracer), listener.NewWriteExecutionLogEventOnExecuteZone(eventsRepo, tracer))

	eventBus.Subscribe(zone.Ignored{
		BasicEvent: cqs.BasicEvent{NameAttr: zone.IgnoredEventName},
	}, listener.NewPublishMessageOnZoneIgnored(chBus, tracer))

	eventBus.Subscribe(zone.FertilizerZoneExecuted{
		BasicEvent: cqs.BasicEvent{NameAttr: zone.FertilizerZoneExecutedEventName},
	}, listener.NewExecutePinsOnExecuteFertilizerZone(chBus, tracer, log), listener.NewWriteExecutionLogEventOnExecuteFertilizerZone(eventsRepo, tracer))

	if err = initStatus(ctx, chBus, qhBus); err != nil {
		log.ErrorContext(ctx, "failed initializing status", slog.String("error", err.Error()))
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	errCh := make(chan error)
	defer close(errCh)

	go updateStatusWorker(ctx, qhBus, chBus, log)
	go eventsWorker(ctx, eventsCh, eventBus, log, tracer)
	go executionInTimeWorker(ctx, qhBus, chBus, log, tracer)

	go func() {
		if err := runTelegramBot(ctx, conf, qhBus, chBus, log, tracer); err != nil {
			log.ErrorContext(ctx, "failed running telegram bot", slog.String("error", err.Error()))
			errCh <- err
		}
	}()
	go func() {
		if err := runHTTPServer(ctx, chBus, qhBus, conf, log, tracer); err != nil {
			log.ErrorContext(ctx, "failed running http server", slog.String("error", err.Error()))
			errCh <- err
		}
	}()
	if err := <-errCh; err != nil {
		return err
	}
	return nil
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
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.ErrorContext(ctx, "[WORKER] Terrace Weather: failed casting weather", slog.String("type", fmt.Sprintf("%T", result)))
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

func buildCron() (*cron.Cron, error) {
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		return nil, err
	}
	c := cron.New(cron.WithLocation(loc))
	return c, nil
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
				if err = publishEvent(ctx, eventsRepo, publisher, &ev, span); err != nil {
					span.RecordError(err)
					log.ErrorContext(ctx, "[WORKER] Reading Events: failed publishing event",
						slog.String("event_id", ev.ID),
						slog.String("error", err.Error()))
				}
				span.End()
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

func runTelegramBot(ctx context.Context, conf *config.Config, qhBus app.QueryBus, chBus app.CommandBus, log *slog.Logger, tracer trace.Tracer) error {
	if !conf.TelegramBotEnabled {
		log.InfoContext(ctx, "[TELEGRAM SERVICE] disabled")
		return nil
	}
	telegramServer, err := telegram.NewCommandReader(conf.TelegramToken, qhBus, chBus, tracer)
	if err != nil {
		log.ErrorContext(ctx, "[TELEGRAM SERVICE] failed building telegram server", slog.String("error", err.Error()))
		return err
	}
	telegramServer.Read(ctx, log)
	return nil
}

func runHTTPServer(ctx context.Context, chBus app.CommandBus, qhBus app.QueryBus, conf *config.Config, log *slog.Logger, tracer trace.Tracer) error {
	definitions := handlersDefinition(chBus, qhBus, conf.AuthToken, tracer)
	httpHandlers := infrahttp.NewHandler(definitions)
	if err := infrahttp.RunServer(ctx, conf.ServerURL, httpHandlers, &infrahttp.CORSOpt{}, log); err != nil {
		log.ErrorContext(ctx, "[HTTP SERVER] failed running server", slog.String("error", err.Error()))
		return err
	}
	return nil
}

func buildLog() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	log := slog.New(handler)
	log.With("service", serviceName)
	return log
}

func executionInTimeWorker(ctx context.Context, qh cqs.QueryHandler, ch cqs.CommandHandler, logger *slog.Logger, tracer trace.Tracer) {
	logger.InfoContext(ctx, "[WORKER] Execution in time: started")
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "[WORKER] Execution in time: context done")
			return
		case <-ticker.C:
			if err := worker.ExecutionInTime(ctx, qh, ch, now(), tracer); err != nil {
				logger.ErrorContext(ctx, "[WORKER] failed execution in time", slog.String("error", err.Error()))
			}
		}
	}
}

func now() time.Time {
	loc, _ := time.LoadLocation("Europe/Madrid")
	return time.Now().In(loc)
}

func eventsWorker(ctx context.Context, ch <-chan tracing.Event, evBus cqs.EventBus, logger *slog.Logger, tracer trace.Tracer) {
	logger.InfoContext(ctx, "[WORKER] Events: started")
	for {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "[WORKER] Events: context done")
			return
		case event := <-ch:
			parentCtx := trace.ContextWithSpanContext(ctx, event.SpanContext)
			ctx, span := tracer.Start(parentCtx, "eventDispatch")
			span.SetAttributes(attribute.String("event-name", event.Event.EventName()))
			if err := evBus.Dispatch(ctx, event.Event); err != nil {
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

func handlersDefinition(chBus app.CommandBus, qhBus app.QueryBus, authToken string, tracer trace.Tracer) infrahttp.HandlersDefinition {
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
			HandlerFunc: authMdw(infrahttp.FinZones(qhBus, tracer)),
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

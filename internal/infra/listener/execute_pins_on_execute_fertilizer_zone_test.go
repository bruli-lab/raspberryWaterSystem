//go:build infra

package listener_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/fake"
	"github.com/bruli/raspberryWaterSystem/internal/infra/listener"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func Test_executePinsOnExecuteFertilizerZone_Listen(t *testing.T) {
	log := buildLog()
	tr := tracer()
	execCh := app.NewExecutePins(fake.NewPinsExecutor(), statusRepository(t, false), tr)
	logCH := &cqs.CommandHandlerMock{}
	logCH.HandleFunc = func(ctx context.Context, _ cqs.Command) ([]event.Event, error) {
		log.DebugContext(ctx, "log command handler")
		return nil, nil
	}
	publishCH := &cqs.CommandHandlerMock{}
	publishCH.HandleFunc = func(ctx context.Context, _ cqs.Command) ([]event.Event, error) {
		log.DebugContext(ctx, "publish message command handler")
		return nil, nil
	}
	chBus := cqs.NewCommandBus()
	chBus.Subscribe(app.ExecutePinsCmdName, execCh)
	chBus.Subscribe(app.SaveExecutionLogCmdName, logCH)
	chBus.Subscribe(app.PublishMessageCmdName, publishCH)
	list := listener.NewExecutePinsOnExecuteFertilizerZone(chBus, tr, log)
	executed := zone.FertilizerZoneExecuted{
		BasicEvent:               event.NewBasicEvent("event-test", uuid.New(), uuid.NewString()),
		ZoneID:                   "bbf",
		ZoneName:                 "bonsai big with fertilizer",
		ZoneSeconds:              10,
		StabilizationZoneSeconds: 3,
		ZoneRelayPins:            []string{"1", "2"},
		CleanValveSeconds:        15,
		CleanValveRelayPin:       "3",
		FertilizerPumpSeconds:    10,
		FertilizerPumpRelayPin:   "4",
		AirZoneSeconds:           15,
		AirZoneRelayPin:          "5",
		FertilizerValveSeconds:   10,
		FertilizerValveRelayPin:  "6",
	}
	err := list.Listen(t.Context(), executed)
	require.NoError(t, err)
}

func buildLog() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	log := slog.New(handler)
	return log
}

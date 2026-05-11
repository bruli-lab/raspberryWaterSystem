//go:build infra

package listener_test

import (
	"context"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/fake"
	"github.com/bruli/raspberryWaterSystem/internal/infra/listener"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func Test_executePinsOnExecuteCleanZone_Listen(t *testing.T) {
	log := buildLog()
	tr := tracer()
	execCh := app.NewExecutePins(fake.NewPinsExecutor(), tr)
	publishCH := &CommandHandlerMock{}
	publishCH.HandleFunc = func(ctx context.Context, cmd cqs.Command) ([]cqs.Event, error) {
		log.DebugContext(ctx, "publish message command handler")
		return nil, nil
	}
	activateCh := &CommandHandlerMock{}
	activateCh.HandleFunc = func(ctx context.Context, cmd cqs.Command) ([]cqs.Event, error) {
		log.DebugContext(ctx, "activate deactivate server command handler")
		return nil, nil
	}
	chBus := app.NewCommandBus()
	chBus.Subscribe(app.ExecutePinsCmdName, execCh)
	chBus.Subscribe(app.PublishMessageCmdName, publishCH)
	chBus.Subscribe(app.ActivateDeactivateServerCmdName, activateCh)
	list := listener.NewExecutePinsOnExecuteCleanZone(chBus, tr, log)
	seconds := uint(10)
	event := zone.CleanZoneExecuted{
		BasicEvent:                cqs.NewBasicEvent("event-test", uuid.New(), uuid.NewString()),
		ZoneID:                    "bbf",
		ZoneName:                  "bonsai big with fertilizer",
		ZoneSeconds:               seconds,
		StabilizationZoneSeconds:  3,
		ZoneRelayPins:             []string{"1", "2"},
		FertilizerPumpSeconds:     seconds,
		FertilizerPumpRelayPin:    "4",
		FertilizerValvuleSeconds:  seconds,
		FertilizerValvuleRelayPin: "6",
		WaitDuration:              20 * time.Second,
		ClarifyDuration:           15 * time.Second,
	}
	err := list.Listen(t.Context(), event)
	require.NoError(t, err)
}

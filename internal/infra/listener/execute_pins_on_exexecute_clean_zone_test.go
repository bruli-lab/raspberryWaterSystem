//go:build infra

package listener_test

import (
	"context"
	"testing"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/fake"
	"github.com/bruli/raspberryWaterSystem/internal/infra/listener"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func Test_executePinsOnExecuteCleanZone_Listen(t *testing.T) {
	log := buildLog()
	tr := tracer()
	execCh := app.NewExecutePins(fake.NewPinsExecutor(), statusRepository(t, true), tr)
	publishCH := &cqs.CommandHandlerMock{}
	publishCH.HandleFunc = func(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
		log.DebugContext(ctx, "publish message command handler")
		return nil, nil
	}
	cleanFinished := make(chan struct{})
	cleanCh := &cqs.CommandHandlerMock{}
	cleanCh.HandleFunc = func(ctx context.Context, _ cqs.Command) ([]event.Event, error) {
		log.DebugContext(ctx, "activate deactivate clean statu command handler")
		close(cleanFinished)
		return nil, nil
	}
	chBus := cqs.NewCommandBus()
	chBus.Subscribe(app.ExecutePinsCmdName, execCh)
	chBus.Subscribe(app.PublishMessageCmdName, publishCH)
	chBus.Subscribe(app.ChangeCleanStatusCmdName, cleanCh)
	list := listener.NewExecutePinsOnExecuteCleanZone(chBus, tr, log)
	seconds := uint(10)
	executed := zone.CleanZoneExecuted{
		BasicEvent:                event.NewBasicEvent("event-test", uuid.New(), uuid.NewString()),
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
	err := list.Listen(t.Context(), executed)
	require.NoError(t, err)
	select {
	case <-cleanFinished:
	case <-time.After(2 * time.Minute):
		t.Fatal("clean cycle did not finish")
	}
	require.Len(t, cleanCh.HandleCalls(), 1)
	require.Len(t, publishCH.HandleCalls(), 2)
}

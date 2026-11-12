package listener_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/listener"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestExecutePinsOnExecuteCleanZoneWhenNotificationsFail(t *testing.T) {
	t.Run(`Given an ExecutePinsOnExecuteCleanZone listener,
	when Listen method is called and publishing notifications returns an error,
	then it runs the clean and clarify phases and clears the clean status`, func(t *testing.T) {
		t.Parallel()
		var (
			m           sync.Mutex
			pinsCalls   int
			cleanStates []bool
		)
		pinsCH := &CommandHandlerMock{
			HandleFunc: func(_ context.Context, _ cqs.Command) ([]cqs.Event, error) {
				m.Lock()
				defer m.Unlock()
				pinsCalls++
				return nil, nil
			},
		}
		publishCH := &CommandHandlerMock{
			HandleFunc: func(_ context.Context, _ cqs.Command) ([]cqs.Event, error) {
				return nil, errors.New("ntfy unreachable")
			},
		}
		cleanCH := &CommandHandlerMock{
			HandleFunc: func(_ context.Context, cmd cqs.Command) ([]cqs.Event, error) {
				co, _ := cmd.(app.ChangeCleanStatusCmd)
				m.Lock()
				defer m.Unlock()
				cleanStates = append(cleanStates, co.Clean)
				return nil, nil
			},
		}
		chBus := app.NewCommandBus()
		chBus.Subscribe(app.ExecutePinsCmdName, pinsCH)
		chBus.Subscribe(app.PublishMessageCmdName, publishCH)
		chBus.Subscribe(app.ChangeCleanStatusCmdName, cleanCH)
		list := listener.NewExecutePinsOnExecuteCleanZone(chBus, tracer(), slog.New(slog.DiscardHandler))
		event := zone.CleanZoneExecuted{
			BasicEvent:                cqs.NewBasicEvent(zone.CleanZoneExecutedEventName, uuid.New(), uuid.NewString()),
			ZoneID:                    "bbf",
			ZoneName:                  "bonsai big with fertilizer",
			ZoneSeconds:               1,
			ZoneRelayPins:             []string{"1", "2"},
			FertilizerPumpSeconds:     1,
			FertilizerPumpRelayPin:    "4",
			FertilizerValvuleSeconds:  1,
			FertilizerValvuleRelayPin: "6",
			WaitDuration:              time.Millisecond,
			ClarifyDuration:           time.Second,
		}

		err := list.Listen(t.Context(), event)

		require.NoError(t, err)
		require.Equal(t, 6, pinsCalls)
		require.Equal(t, []bool{true, false}, cleanStates)
		require.Len(t, publishCH.HandleCalls(), 2)
	})
}

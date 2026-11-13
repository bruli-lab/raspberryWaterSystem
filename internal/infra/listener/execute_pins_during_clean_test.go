package listener_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/listener"
	"github.com/bruli/raspberryWaterSystem/internal/infra/memory"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type countingPinsExecutor struct {
	calls atomic.Int32
}

func (c *countingPinsExecutor) Execute(_ context.Context, _ uint, _ []string) error {
	c.calls.Add(1)
	return nil
}

func TestExecutePinsOnExecuteZoneDuringClean(t *testing.T) {
	tests := []struct {
		name               string
		clean              bool
		expectedErr        error
		expectedExecutions int32
		expectedLogs       int
	}{
		{
			name:               "and no clean cycle is in progress, then it opens the zone valves and saves the execution log",
			expectedExecutions: 1,
			expectedLogs:       1,
		},
		{
			name:        "and a clean cycle is in progress, then it discards the irrigation without opening any valve",
			clean:       true,
			expectedErr: app.CleanInProgressError{},
		},
	}
	for _, tt := range tests {
		t.Run(`Given an ExecutePinsOnExecuteZone listener with a queued zone executed event,
		when Listen method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			statusRepo := memory.NewStatusRepository(tracer())
			require.NoError(t, statusRepo.Save(t.Context(), fixtures.StatusBuilder{Active: true, Clean: tt.clean}.Build()))
			pins := &countingPinsExecutor{}
			logCH := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
					return nil, nil
				},
			}
			publishCH := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
					return nil, nil
				},
			}
			chBus := cqs.NewCommandBus()
			chBus.Subscribe(app.ExecutePinsCmdName, app.NewExecutePins(pins, statusRepo, tracer()))
			chBus.Subscribe(app.SaveExecutionLogCmdName, logCH)
			chBus.Subscribe(app.PublishMessageCmdName, publishCH)
			list := listener.NewExecutePinsOnExecuteZone(chBus, tracer())
			executed := zone.Executed{
				BasicEvent: event.NewBasicEvent(zone.ExecutedEventName, uuid.New(), uuid.NewString()),
				ZoneID:     "zone-a",
				ZoneName:   "zone a",
				Seconds:    10,
				RelayPins:  []string{"18"},
			}

			err := list.Listen(t.Context(), executed)

			require.Equal(t, tt.expectedExecutions, pins.calls.Load())
			require.Len(t, logCH.HandleCalls(), tt.expectedLogs)
			if tt.expectedErr != nil {
				require.IsType(t, tt.expectedErr, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

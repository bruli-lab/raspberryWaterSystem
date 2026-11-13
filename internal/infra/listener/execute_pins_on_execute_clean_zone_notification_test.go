package listener_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/listener"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type cleanCycleRecorder struct {
	m           sync.Mutex
	pinsCalls   int
	cleanStates []bool
}

func (r *cleanCycleRecorder) pins() int {
	r.m.Lock()
	defer r.m.Unlock()
	return r.pinsCalls
}

func (r *cleanCycleRecorder) states() []bool {
	r.m.Lock()
	defer r.m.Unlock()
	return append([]bool(nil), r.cleanStates...)
}

func cleanZoneEvent(wait time.Duration) zone.CleanZoneExecuted {
	return zone.CleanZoneExecuted{
		BasicEvent:              event.NewBasicEvent(zone.CleanZoneExecutedEventName, uuid.New(), uuid.NewString()),
		ZoneID:                  "bbf",
		ZoneName:                "bonsai big with fertilizer",
		ZoneSeconds:             1,
		ZoneRelayPins:           []string{"1", "2"},
		FertilizerPumpSeconds:   1,
		FertilizerPumpRelayPin:  "4",
		FertilizerValveSeconds:  1,
		FertilizerValveRelayPin: "6",
		WaitDuration:            wait,
		ClarifyDuration:         time.Second,
	}
}

func TestExecutePinsOnExecuteCleanZoneInBackground(t *testing.T) {
	tests := []struct {
		name                 string
		event                event.Event
		publishErr           error
		cancelAfter          time.Duration
		expectedErr          bool
		expectedPinsCalls    int
		expectedPublishCalls int
	}{
		{
			name:                 "and publishing notifications fails, then it returns immediately, runs clean and clarify in the background and clears the clean status",
			event:                cleanZoneEvent(time.Millisecond),
			publishErr:           errors.New("ntfy unreachable"),
			expectedPinsCalls:    6,
			expectedPublishCalls: 2,
		},
		{
			name:                 "and the context is cancelled while waiting for the clarify phase, then it stops the cycle and clears the clean status",
			event:                cleanZoneEvent(time.Hour),
			cancelAfter:          100 * time.Millisecond,
			expectedPinsCalls:    3,
			expectedPublishCalls: 1,
		},
		{
			name:        "with a wrong event, then it returns an error and does not change the clean status",
			event:       zone.Ignored{},
			expectedErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an ExecutePinsOnExecuteCleanZone listener,
		when Listen method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			recorder := &cleanCycleRecorder{}
			pinsCH := &cqs.CommandHandlerMock{
				HandleFunc: func(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
					recorder.m.Lock()
					recorder.pinsCalls++
					recorder.m.Unlock()
					co, _ := cmd.(app.ExecutePinsCmd)
					timer := time.NewTimer(time.Duration(co.Seconds) * time.Millisecond)
					defer timer.Stop()
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-timer.C:
						return nil, nil
					}
				},
			}
			publishCH := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
					return nil, tt.publishErr
				},
			}
			cleanCH := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, cmd cqs.Command) ([]event.Event, error) {
					co, _ := cmd.(app.ChangeCleanStatusCmd)
					recorder.m.Lock()
					defer recorder.m.Unlock()
					recorder.cleanStates = append(recorder.cleanStates, co.Clean)
					return nil, nil
				},
			}
			chBus := cqs.NewCommandBus()
			chBus.Subscribe(app.ExecutePinsCmdName, pinsCH)
			chBus.Subscribe(app.PublishMessageCmdName, publishCH)
			chBus.Subscribe(app.ChangeCleanStatusCmdName, cleanCH)
			list := listener.NewExecutePinsOnExecuteCleanZone(chBus, listener.WaitFor, tracer(), slog.New(slog.DiscardHandler))
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tt.cancelAfter > 0 {
				time.AfterFunc(tt.cancelAfter, cancel)
			}

			started := time.Now()
			err := list.Listen(ctx, tt.event)

			require.Less(t, time.Since(started), 50*time.Millisecond)
			if tt.expectedErr {
				require.Error(t, err)
				require.Never(t, func() bool { return len(recorder.states()) > 0 }, 100*time.Millisecond, 10*time.Millisecond)
				return
			}
			require.NoError(t, err)
			require.Eventually(t, func() bool {
				return len(recorder.states()) == 1
			}, 5*time.Second, 10*time.Millisecond)
			require.Equal(t, []bool{false}, recorder.states())
			require.Equal(t, tt.expectedPinsCalls, recorder.pins())
			require.Len(t, publishCH.HandleCalls(), tt.expectedPublishCalls)
		})
	}
}

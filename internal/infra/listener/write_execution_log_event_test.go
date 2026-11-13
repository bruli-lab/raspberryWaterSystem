package listener_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/listener"
	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestWriteExecutionLogEventListeners(t *testing.T) {
	now := time.Date(2026, time.November, 13, 9, 30, 0, 0, time.UTC)
	tracer := noop.NewTracerProvider().Tracer("test")
	executed := zone.Executed{
		BasicEvent: event.NewBasicEvent(zone.ExecutedEventName, uuid.New(), "zone-a"),
		ZoneName:   "zone a",
		Seconds:    20,
	}
	fertilizerExecuted := zone.FertilizerZoneExecuted{
		BasicEvent:  event.NewBasicEvent(zone.FertilizerZoneExecutedEventName, uuid.New(), "zone-b"),
		ZoneName:    "zone b",
		ZoneSeconds: 30,
	}
	errSave := errors.New("disk full")
	tests := []struct {
		name            string
		build           func(listener.OutboxEvents) event.Listener
		ev              event.Event
		saveErr         error
		expectedErr     error
		expectedPayload outbox.ExecutionLog
	}{
		{
			name: "Given a WriteExecutionLogEventOnExecuteZone listener, when Listen method is called and save returns an error, then it returns same error",
			build: func(events listener.OutboxEvents) event.Listener {
				return listener.NewWriteExecutionLogEventOnExecuteZone(events, func() time.Time { return now }, tracer)
			},
			ev:          executed,
			saveErr:     errSave,
			expectedErr: errSave,
		},
		{
			name: "Given a WriteExecutionLogEventOnExecuteZone listener, when Listen method is called, then it saves an execution log event with the zone seconds",
			build: func(events listener.OutboxEvents) event.Listener {
				return listener.NewWriteExecutionLogEventOnExecuteZone(events, func() time.Time { return now }, tracer)
			},
			ev:              executed,
			expectedPayload: outbox.ExecutionLog{Seconds: 20, ZoneName: "zone a", ExecutedAt: executed.EventAt()},
		},
		{
			name: "Given a WriteExecutionLogEventOnExecuteFertilizerZone listener, when Listen method is called, then it saves an execution log event with the zone seconds",
			build: func(events listener.OutboxEvents) event.Listener {
				return listener.NewWriteExecutionLogEventOnExecuteFertilizerZone(events, func() time.Time { return now }, tracer)
			},
			ev:              fertilizerExecuted,
			expectedPayload: outbox.ExecutionLog{Seconds: 30, ZoneName: "zone b", ExecutedAt: fertilizerExecuted.EventAt()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			events := &OutboxEventsMock{
				SaveFunc: func(_ context.Context, _ *outbox.Event) error {
					return tt.saveErr
				},
			}
			err := tt.build(events).Listen(t.Context(), tt.ev)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, events.SaveCalls(), 1)
			saved := events.SaveCalls()[0].Ev
			require.Equal(t, outbox.ExecutionLogsEventType, saved.EventType)
			require.Equal(t, now, saved.EventAt)
			var payload outbox.ExecutionLog
			require.NoError(t, json.Unmarshal(saved.Payload, &payload))
			require.True(t, tt.expectedPayload.ExecutedAt.Equal(payload.ExecutedAt))
			payload.ExecutedAt = tt.expectedPayload.ExecutedAt
			require.Equal(t, tt.expectedPayload, payload)
		})
	}
}

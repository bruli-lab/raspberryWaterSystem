package worker_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"github.com/bruli/raspberryWaterSystem/internal/infra/worker"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

type pendingEvents struct {
	events  []outbox.Event
	findErr error
	removed []string
}

func (p *pendingEvents) FindAll(_ context.Context) ([]outbox.Event, error) {
	return p.events, p.findErr
}

func (p *pendingEvents) Remove(_ context.Context, ev *outbox.Event) error {
	p.removed = append(p.removed, ev.ID)
	return nil
}

type eventPublisher struct {
	connected        bool
	failingIDs       map[string]bool
	disconnectsOnErr bool
	published        []string
}

func (e *eventPublisher) IsConnected() bool {
	return e.connected
}

func (e *eventPublisher) PublishEvent(_ context.Context, id, _ string, _ []byte) error {
	if e.failingIDs[id] {
		if e.disconnectsOnErr {
			e.connected = false
		}
		return errors.New("publish failed")
	}
	e.published = append(e.published, id)
	return nil
}

func TestPublishOutboxEvents(t *testing.T) {
	events := []outbox.Event{
		{ID: "1", EventType: outbox.ExecutionLogsEventType},
		{ID: "2", EventType: outbox.TerraceWeatherEventType},
		{ID: "3", EventType: outbox.ExecutionLogsEventType},
	}
	errFind := errors.New("unreadable directory")
	tests := []struct {
		name              string
		publisher         *eventPublisher
		findErr           error
		expectedPublished []string
		expectedRemoved   []string
		expectedErr       error
		expectedErrAs     any
	}{
		{
			name:          "and the publisher is not connected, then it keeps every event and returns a not connected error",
			publisher:     &eventPublisher{},
			expectedErrAs: &worker.PublisherNotConnectedError{},
		},
		{
			name:        "and FindAll returns an error, then it returns same error",
			publisher:   &eventPublisher{connected: true},
			findErr:     errFind,
			expectedErr: errFind,
		},
		{
			name:              "and every event is published, then it removes all of them in order",
			publisher:         &eventPublisher{connected: true},
			expectedPublished: []string{"1", "2", "3"},
			expectedRemoved:   []string{"1", "2", "3"},
		},
		{
			name:              "and one event fails while still connected, then it keeps that event and publishes the rest",
			publisher:         &eventPublisher{connected: true, failingIDs: map[string]bool{"2": true}},
			expectedPublished: []string{"1", "3"},
			expectedRemoved:   []string{"1", "3"},
			expectedErrAs:     &worker.PublishOutboxEventError{},
		},
		{
			name:              "and the connection drops while publishing, then it stops and keeps the remaining events",
			publisher:         &eventPublisher{connected: true, failingIDs: map[string]bool{"2": true}, disconnectsOnErr: true},
			expectedPublished: []string{"1"},
			expectedRemoved:   []string{"1"},
			expectedErrAs:     &worker.PublisherNotConnectedError{},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a PublishOutboxEvents worker, when it runs `+tt.name, func(t *testing.T) {
			t.Parallel()
			pending := &pendingEvents{events: events, findErr: tt.findErr}
			published, err := worker.PublishOutboxEvents(t.Context(), pending, tt.publisher, noop.NewTracerProvider().Tracer("test"))
			switch {
			case tt.expectedErr != nil:
				require.ErrorIs(t, err, tt.expectedErr)
			case tt.expectedErrAs != nil:
				require.ErrorAs(t, err, tt.expectedErrAs)
			default:
				require.NoError(t, err)
			}
			require.Equal(t, len(tt.expectedPublished), published)
			require.Equal(t, tt.expectedPublished, tt.publisher.published)
			require.Equal(t, tt.expectedRemoved, pending.removed)
		})
	}
}

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

const defaultTestBatchSize = 10

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
	batches          [][]string
}

func (e *eventPublisher) IsConnected() bool {
	return e.connected
}

func (e *eventPublisher) PublishEvents(_ context.Context, events []outbox.Event) []error {
	ids := make([]string, len(events))
	for i, ev := range events {
		ids[i] = ev.ID
	}
	e.batches = append(e.batches, ids)

	errs := make([]error, len(events))
	for i, ev := range events {
		switch {
		case !e.connected:
			errs[i] = errors.New("not connected")
		case e.failingIDs[ev.ID]:
			if e.disconnectsOnErr {
				e.connected = false
			}
			errs[i] = errors.New("publish failed")
		default:
			e.published = append(e.published, ev.ID)
		}
	}
	return errs
}

func TestPublishOutboxEvents(t *testing.T) {
	events := []outbox.Event{
		{ID: "1", EventType: outbox.ExecutionLogsEventType},
		{ID: "2", EventType: outbox.TerraceWeatherEventType},
		{ID: "3", EventType: outbox.ExecutionLogsEventType},
	}
	fourEvents := []outbox.Event{
		{ID: "1", EventType: outbox.ExecutionLogsEventType},
		{ID: "2", EventType: outbox.TerraceWeatherEventType},
		{ID: "3", EventType: outbox.ExecutionLogsEventType},
		{ID: "4", EventType: outbox.ExecutionLogsEventType},
	}
	errFind := errors.New("unreadable directory")
	tests := []struct {
		name              string
		events            []outbox.Event
		publisher         *eventPublisher
		findErr           error
		batchSize         int
		expectedPublished []string
		expectedRemoved   []string
		expectedBatches   [][]string
		expectedErr       error
		expectedErrAs     any
	}{
		{
			name:          "and the publisher is not connected, then it keeps every event and returns a not connected error",
			events:        events,
			publisher:     &eventPublisher{},
			expectedErrAs: &worker.PublisherNotConnectedError{},
		},
		{
			name:        "and FindAll returns an error, then it returns same error",
			events:      events,
			publisher:   &eventPublisher{connected: true},
			findErr:     errFind,
			expectedErr: errFind,
		},
		{
			name:              "and every event is published, then it removes all of them in order",
			events:            events,
			publisher:         &eventPublisher{connected: true},
			expectedPublished: []string{"1", "2", "3"},
			expectedRemoved:   []string{"1", "2", "3"},
			expectedBatches:   [][]string{{"1", "2", "3"}},
		},
		{
			name:              "and one event fails while still connected, then it keeps that event and publishes the rest",
			events:            events,
			publisher:         &eventPublisher{connected: true, failingIDs: map[string]bool{"2": true}},
			expectedPublished: []string{"1", "3"},
			expectedRemoved:   []string{"1", "3"},
			expectedErrAs:     &worker.PublishOutboxEventError{},
		},
		{
			name:              "and the connection drops while publishing, then it keeps the remaining events of that batch",
			events:            events,
			publisher:         &eventPublisher{connected: true, failingIDs: map[string]bool{"2": true}, disconnectsOnErr: true},
			expectedPublished: []string{"1"},
			expectedRemoved:   []string{"1"},
			expectedErrAs:     &worker.PublisherNotConnectedError{},
		},
		{
			name:              "and there are more events than the batch size, then it publishes them in more than one batch",
			events:            fourEvents,
			publisher:         &eventPublisher{connected: true},
			batchSize:         2,
			expectedPublished: []string{"1", "2", "3", "4"},
			expectedRemoved:   []string{"1", "2", "3", "4"},
			expectedBatches:   [][]string{{"1", "2"}, {"3", "4"}},
		},
		{
			name:   "and the connection drops in the first batch, then it never starts the next batch",
			events: fourEvents,
			publisher: &eventPublisher{
				connected: true, failingIDs: map[string]bool{"2": true}, disconnectsOnErr: true,
			},
			batchSize:         2,
			expectedPublished: []string{"1"},
			expectedRemoved:   []string{"1"},
			expectedBatches:   [][]string{{"1", "2"}},
			expectedErrAs:     &worker.PublisherNotConnectedError{},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a PublishOutboxEvents worker, when it runs `+tt.name, func(t *testing.T) {
			t.Parallel()
			pending := &pendingEvents{events: tt.events, findErr: tt.findErr}
			batchSize := tt.batchSize
			if batchSize == 0 {
				batchSize = defaultTestBatchSize
			}
			published, err := worker.PublishOutboxEvents(t.Context(), pending, tt.publisher, noop.NewTracerProvider().Tracer("test"), batchSize)
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
			if tt.expectedBatches != nil {
				require.Equal(t, tt.expectedBatches, tt.publisher.batches)
			}
		})
	}
}

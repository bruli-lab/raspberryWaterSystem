package worker_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/bruli/raspberryWaterSystem/internal/infra/worker"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type memoryPendingStore struct {
	m        sync.Mutex
	saveErr  error
	findErr  error
	pendings map[string]disk.PendingIrrigation
	order    []string
	removed  []string
}

func newMemoryPendingStore(pendings ...disk.PendingIrrigation) *memoryPendingStore {
	s := &memoryPendingStore{pendings: map[string]disk.PendingIrrigation{}}
	for _, p := range pendings {
		s.pendings[p.ID] = p
		s.order = append(s.order, p.ID)
	}
	return s
}

func (s *memoryPendingStore) Save(_ context.Context, ev event.Event) (disk.PendingIrrigation, error) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.saveErr != nil {
		return disk.PendingIrrigation{}, s.saveErr
	}
	p := disk.PendingIrrigation{ID: strconv.Itoa(len(s.order) + 1), EnqueuedAt: time.Now(), Event: ev}
	s.pendings[p.ID] = p
	s.order = append(s.order, p.ID)
	return p, nil
}

func (s *memoryPendingStore) Remove(_ context.Context, id string) error {
	s.m.Lock()
	defer s.m.Unlock()
	delete(s.pendings, id)
	s.removed = append(s.removed, id)
	return nil
}

func (s *memoryPendingStore) FindAll(_ context.Context) ([]disk.PendingIrrigation, error) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.findErr != nil {
		return nil, s.findErr
	}
	out := make([]disk.PendingIrrigation, 0, len(s.order))
	for _, id := range s.order {
		if p, ok := s.pendings[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *memoryPendingStore) stored() int {
	s.m.Lock()
	defer s.m.Unlock()
	return len(s.pendings)
}

func zoneExecuted(name string) zone.Executed {
	return zone.Executed{
		BasicEvent: event.NewBasicEvent(zone.ExecutedEventName, uuid.New(), name),
		ZoneName:   name,
	}
}

func zoneIgnored(name string) zone.Ignored {
	return zone.Ignored{
		BasicEvent: event.NewBasicEvent(zone.IgnoredEventName, uuid.New(), name),
		ZoneName:   name,
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func receive(t *testing.T, out <-chan app.TracedEvent) event.Event {
	t.Helper()
	select {
	case ev := <-out:
		return ev.Event
	case <-time.After(2 * time.Second):
		require.FailNow(t, "event not forwarded")
		return nil
	}
}

func TestQueueEvents(t *testing.T) {
	tests := []struct {
		name              string
		events            int
		irrigation        bool
		saveErr           error
		expectedPersisted int
	}{
		{
			name:              "with more irrigations than the consumer can take while it is busy, then producers never block and every irrigation is persisted",
			events:            200,
			irrigation:        true,
			expectedPersisted: 200,
		},
		{
			name:              "with events that are not irrigations, then they are forwarded without being persisted",
			events:            3,
			expectedPersisted: 0,
		},
		{
			name:              "with irrigations that cannot be persisted, then they are still forwarded",
			events:            3,
			irrigation:        true,
			saveErr:           errors.New("disk full"),
			expectedPersisted: 0,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an events queue with a busy consumer,
		when events are sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			store := newMemoryPendingStore()
			store.saveErr = tt.saveErr
			in := make(chan app.TracedEvent)
			out := make(chan app.TracedEvent)
			go worker.QueueEvents(ctx, in, out, store, nil, discardLogger())

			sent := make(chan struct{})
			go func() {
				for i := range tt.events {
					name := strconv.Itoa(i)
					var ev event.Event = zoneIgnored(name)
					if tt.irrigation {
						ev = zoneExecuted(name)
					}
					in <- app.TracedEvent{Event: ev}
				}
				close(sent)
			}()
			select {
			case <-sent:
			case <-time.After(2 * time.Second):
				require.FailNow(t, "producer blocked while the consumer was busy")
			}
			require.Eventually(t, func() bool { return store.stored() == tt.expectedPersisted }, 2*time.Second, 10*time.Millisecond)

			for i := range tt.events {
				require.Equal(t, strconv.Itoa(i), receive(t, out).AggregateRootID())
			}
			require.Eventually(t, func() bool { return store.stored() == 0 }, 2*time.Second, 10*time.Millisecond)
		})
	}
}

func TestQueueEventsWithRestoredIrrigations(t *testing.T) {
	t.Run(`Given an events queue with restored pending irrigations,
	when a new event is sent,
	then restored irrigations are forwarded first and removed once handed over`, func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		restored := []disk.PendingIrrigation{
			{ID: "a", EnqueuedAt: time.Now(), Event: zoneExecuted("restored 1")},
			{ID: "b", EnqueuedAt: time.Now(), Event: zoneExecuted("restored 2")},
		}
		store := newMemoryPendingStore(restored...)
		in := make(chan app.TracedEvent)
		out := make(chan app.TracedEvent)
		go worker.QueueEvents(ctx, in, out, store, restored, discardLogger())

		in <- app.TracedEvent{Event: zoneIgnored("new")}

		require.Equal(t, "restored 1", receive(t, out).AggregateRootID())
		require.Equal(t, "restored 2", receive(t, out).AggregateRootID())
		require.Equal(t, "new", receive(t, out).AggregateRootID())
		require.Eventually(t, func() bool { return store.stored() == 0 }, 2*time.Second, 10*time.Millisecond)
	})
}

func TestRestorePendingIrrigations(t *testing.T) {
	now := time.Date(2026, time.November, 12, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name            string
		pendings        []disk.PendingIrrigation
		findErr         error
		expectedIDs     []string
		expectedRemoved []string
	}{
		{
			name: "with recent and stale irrigations, then it restores the recent ones and removes the stale ones",
			pendings: []disk.PendingIrrigation{
				{ID: "stale", EnqueuedAt: now.Add(-2 * time.Hour), Event: zoneExecuted("stale")},
				{ID: "recent", EnqueuedAt: now.Add(-10 * time.Minute), Event: zoneExecuted("recent")},
			},
			expectedIDs:     []string{"recent"},
			expectedRemoved: []string{"stale"},
		},
		{
			name:        "and the pending irrigations cannot be read, then it restores nothing without failing",
			findErr:     errors.New("io error"),
			expectedIDs: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(`Given stored pending irrigations,
		when RestorePendingIrrigations is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			store := newMemoryPendingStore(tt.pendings...)
			store.findErr = tt.findErr

			restored := worker.RestorePendingIrrigations(t.Context(), store, time.Hour, now, discardLogger())

			ids := make([]string, 0, len(restored))
			for _, r := range restored {
				ids = append(ids, r.ID)
			}
			require.Equal(t, tt.expectedIDs, ids)
			require.Equal(t, tt.expectedRemoved, store.removed)
		})
	}
}

package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
)

type PendingIrrigationsStore interface {
	Save(ctx context.Context, ev event.Event) (disk.PendingIrrigation, error)
	Remove(ctx context.Context, id string) error
	FindAll(ctx context.Context) ([]disk.PendingIrrigation, error)
}

type queuedEvent struct {
	event     app.TracedEvent
	pendingID string
}

func RestorePendingIrrigations(
	ctx context.Context,
	store PendingIrrigationsStore,
	maxAge time.Duration,
	now time.Time,
	log *slog.Logger,
) []disk.PendingIrrigation {
	pendings, err := store.FindAll(ctx)
	if err != nil {
		log.ErrorContext(ctx, "[QUEUE] failed reading pending irrigations", slog.String("error", err.Error()))
		return nil
	}
	restored := make([]disk.PendingIrrigation, 0, len(pendings))
	for _, pending := range pendings {
		if now.Sub(pending.EnqueuedAt) > maxAge {
			log.WarnContext(ctx, "[QUEUE] discarding stale pending irrigation",
				slog.String("event", pending.Event.EventName()),
				slog.Time("enqueued_at", pending.EnqueuedAt))
			removePendingIrrigation(ctx, store, pending.ID, log)
			continue
		}
		restored = append(restored, pending)
	}
	if len(restored) > 0 {
		log.InfoContext(ctx, "[QUEUE] restoring pending irrigations", slog.Int("count", len(restored)))
	}
	return restored
}

func QueueEvents(
	ctx context.Context,
	in <-chan app.TracedEvent,
	out chan<- app.TracedEvent,
	store PendingIrrigationsStore,
	restored []disk.PendingIrrigation,
	log *slog.Logger,
) {
	pending := make([]queuedEvent, 0, len(restored))
	for _, r := range restored {
		pending = append(pending, queuedEvent{event: app.TracedEvent{Event: r.Event}, pendingID: r.ID})
	}
	for {
		var next queuedEvent
		var sendCh chan<- app.TracedEvent
		if len(pending) > 0 {
			next = pending[0]
			sendCh = out
		}
		select {
		case <-ctx.Done():
			return
		case ev := <-in:
			pending = append(pending, persistEvent(ctx, store, &ev, log))
		case sendCh <- next.event:
			pending[0] = queuedEvent{}
			pending = pending[1:]
			if next.pendingID != "" {
				removePendingIrrigation(ctx, store, next.pendingID, log)
			}
		}
	}
}

func persistEvent(ctx context.Context, store PendingIrrigationsStore, ev *app.TracedEvent, log *slog.Logger) queuedEvent {
	if !disk.IsPendingIrrigation(ev.Event) {
		return queuedEvent{event: *ev}
	}
	saved, err := store.Save(ctx, ev.Event)
	if err != nil {
		log.ErrorContext(ctx, "[QUEUE] failed persisting pending irrigation, it is kept only in memory",
			slog.String("event", ev.Event.EventName()), slog.String("error", err.Error()))
		return queuedEvent{event: *ev}
	}
	return queuedEvent{event: *ev, pendingID: saved.ID}
}

func removePendingIrrigation(ctx context.Context, store PendingIrrigationsStore, id string, log *slog.Logger) {
	if err := store.Remove(ctx, id); err != nil {
		log.ErrorContext(ctx, "[QUEUE] failed removing pending irrigation", slog.String("id", id), slog.String("error", err.Error()))
	}
}

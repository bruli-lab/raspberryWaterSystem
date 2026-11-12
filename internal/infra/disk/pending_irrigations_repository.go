package disk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type PendingIrrigation struct {
	ID         string
	EnqueuedAt time.Time
	Event      event.Event
}

type pendingIrrigationFile struct {
	Name       string          `json:"name"`
	EnqueuedAt time.Time       `json:"enqueued_at"`
	Event      json.RawMessage `json:"event"`
}

func IsPendingIrrigation(ev event.Event) bool {
	switch ev.(type) {
	case zone.Executed, zone.FertilizerZoneExecuted:
		return true
	default:
		return false
	}
}

type PendingIrrigationsRepository struct {
	dir      string
	tracer   trace.Tracer
	sequence *atomic.Uint64
	now      func() time.Time
}

func (p PendingIrrigationsRepository) Save(ctx context.Context, ev event.Event) (PendingIrrigation, error) {
	_, span := p.tracer.Start(ctx, "PendingIrrigationsRepository.Save")
	defer span.End()
	pending, err := p.save(ev)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return PendingIrrigation{}, err
	}
	span.SetStatus(codes.Ok, "pending irrigation saved")
	return pending, nil
}

func (p PendingIrrigationsRepository) save(ev event.Event) (PendingIrrigation, error) {
	if !IsPendingIrrigation(ev) {
		return PendingIrrigation{}, UnsupportedPendingIrrigationError{name: ev.EventName()}
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return PendingIrrigation{}, fmt.Errorf("failed to marshal pending irrigation: %w", err)
	}
	enqueuedAt := p.now()
	id := fmt.Sprintf("%020d-%010d", enqueuedAt.UnixNano(), p.sequence.Add(1))
	if err = os.MkdirAll(p.dir, 0o755); err != nil {
		return PendingIrrigation{}, fmt.Errorf("failed to create pending irrigations directory: %w", err)
	}
	file := pendingIrrigationFile{Name: ev.EventName(), EnqueuedAt: enqueuedAt, Event: payload}
	if err := writeJsonFile(p.path(id), file); err != nil {
		return PendingIrrigation{}, err
	}
	return PendingIrrigation{ID: id, EnqueuedAt: enqueuedAt, Event: ev}, nil
}

func (p PendingIrrigationsRepository) Remove(ctx context.Context, id string) error {
	_, span := p.tracer.Start(ctx, "PendingIrrigationsRepository.Remove")
	defer span.End()
	if err := os.Remove(p.path(id)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		err = fmt.Errorf("failed to remove pending irrigation %s: %w", id, err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "pending irrigation removed")
	return nil
}

func (p PendingIrrigationsRepository) FindAll(ctx context.Context) ([]PendingIrrigation, error) {
	_, span := p.tracer.Start(ctx, "PendingIrrigationsRepository.FindAll")
	defer span.End()
	entries, err := os.ReadDir(p.dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			span.SetStatus(codes.Ok, "pending irrigations directory does not exist yet")
			return nil, nil
		}
		err = fmt.Errorf("failed to read pending irrigations directory: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	pendings := make([]PendingIrrigation, 0, len(entries))
	for _, entry := range entries {
		if !isEventFile(entry) {
			continue
		}
		path := filepath.Join(p.dir, entry.Name())
		pending, err := readPendingIrrigation(path)
		if err != nil {
			span.RecordError(err)
			if err := moveToDeadLetter(p.dir, path); err != nil {
				span.RecordError(err)
			}
			continue
		}
		pendings = append(pendings, pending)
	}
	slices.SortFunc(pendings, func(a, b PendingIrrigation) int {
		return strings.Compare(a.ID, b.ID)
	})
	span.SetStatus(codes.Ok, "pending irrigations found")
	return pendings, nil
}

func readPendingIrrigation(path string) (PendingIrrigation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PendingIrrigation{}, fmt.Errorf("failed to read pending irrigation %s: %w", path, err)
	}
	var file pendingIrrigationFile
	if err = json.Unmarshal(data, &file); err != nil {
		return PendingIrrigation{}, fmt.Errorf("failed to decode pending irrigation %s: %w", path, err)
	}
	ev, err := decodePendingIrrigationEvent(file)
	if err != nil {
		return PendingIrrigation{}, fmt.Errorf("failed to decode pending irrigation %s: %w", path, err)
	}
	id := filepath.Base(path)
	id = id[:len(id)-len(filepath.Ext(id))]
	return PendingIrrigation{ID: id, EnqueuedAt: file.EnqueuedAt, Event: ev}, nil
}

func decodePendingIrrigationEvent(file pendingIrrigationFile) (event.Event, error) {
	switch file.Name {
	case zone.ExecutedEventName:
		var ev zone.Executed
		err := json.Unmarshal(file.Event, &ev)
		return ev, err
	case zone.FertilizerZoneExecutedEventName:
		var ev zone.FertilizerZoneExecuted
		err := json.Unmarshal(file.Event, &ev)
		return ev, err
	default:
		return nil, UnsupportedPendingIrrigationError{name: file.Name}
	}
}

func (p PendingIrrigationsRepository) path(id string) string {
	return filepath.Join(p.dir, id+".json")
}

func NewPendingIrrigationsRepository(dir string, now func() time.Time, tracer trace.Tracer) PendingIrrigationsRepository {
	return PendingIrrigationsRepository{dir: dir, tracer: tracer, sequence: &atomic.Uint64{}, now: now}
}

type UnsupportedPendingIrrigationError struct {
	name string
}

func (u UnsupportedPendingIrrigationError) Error() string {
	return fmt.Sprintf("event %q is not a pending irrigation", u.name)
}

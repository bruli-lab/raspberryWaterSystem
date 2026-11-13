package disk

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const DeadLetterDirectory = "dead-letter"

type EventsRepository struct {
	eventsDir string
	tracer    trace.Tracer
}

func (e EventsRepository) Save(ctx context.Context, ev *outbox.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := e.tracer.Start(ctx, "EventsRepository.Save")
		defer span.End()
		if err := os.MkdirAll(e.eventsDir, DirPermissions); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		if err := writeJsonFile(fmt.Sprintf("%s/%v.json", e.eventsDir, ev.EventAt.UnixMicro()), ev); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.SetStatus(codes.Ok, "event saved")
		return nil
	}
}

func (e EventsRepository) Remove(ctx context.Context, ev *outbox.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := e.tracer.Start(ctx, "EventsRepository.Remove")
		defer span.End()
		unix := ev.EventAt.UnixMicro()
		err := os.Remove(fmt.Sprintf("%s/%v.json", e.eventsDir, unix))
		if err != nil {
			err = fmt.Errorf("failed to remove event file: %w", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.SetStatus(codes.Ok, "event removed")
		return nil
	}
}

func (e EventsRepository) FindAll(ctx context.Context) ([]outbox.Event, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		_, span := e.tracer.Start(ctx, "EventsRepository.FindAll")
		defer span.End()
		files, err := os.ReadDir(e.eventsDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				span.SetStatus(codes.Ok, "events directory does not exist yet")
				return nil, nil
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("failed to read events directory: %w", err)
		}
		events := make([]outbox.Event, 0, len(files))
		for _, f := range files {
			if !isEventFile(f) {
				continue
			}
			path := filepath.Join(e.eventsDir, f.Name())
			var ev outbox.Event
			if err := readJsonFile(path, &ev); err != nil {
				span.RecordError(fmt.Errorf("failed to read event file %s: %w", f.Name(), err))
				if err := e.moveToDeadLetter(path); err != nil {
					span.RecordError(err)
				}
				continue
			}
			events = append(events, ev)
		}
		span.SetStatus(codes.Ok, "events found")
		return events, nil
	}
}

func (e EventsRepository) moveToDeadLetter(path string) error {
	return moveToDeadLetter(e.eventsDir, path)
}

func moveToDeadLetter(dir, path string) error {
	deadLetterDir := filepath.Join(dir, DeadLetterDirectory)
	if err := os.MkdirAll(deadLetterDir, DirPermissions); err != nil {
		return fmt.Errorf("failed to create dead letter directory: %w", err)
	}
	if err := os.Rename(path, filepath.Join(deadLetterDir, filepath.Base(path))); err != nil {
		return fmt.Errorf("failed to move event file to dead letter directory: %w", err)
	}
	return nil
}

func isEventFile(f fs.DirEntry) bool {
	name := f.Name()
	return f.Type().IsRegular() && filepath.Ext(name) == ".json" && !strings.HasPrefix(name, tempFilePrefix)
}

func NewEventsRepository(eventsDir string, tracer trace.Tracer) *EventsRepository {
	return &EventsRepository{eventsDir: eventsDir, tracer: tracer}
}

package disk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	ExecutionLogsEventType  = "execution_logs"
	TerraceWeatherEventType = "terrace_weather"
	DeadLetterDirectory     = "dead-letter"
)

type Event struct {
	ID        string            `json:"id"`
	EventType string            `json:"event_type"`
	EventAt   time.Time         `json:"event_at"`
	Payload   []byte            `json:"payload"`
	Trace     map[string]string `json:"trace"`
}

type Weather struct {
	Temperature float32 `json:"temperature"`
	IsRaining   bool    `json:"is_raining"`
	Humidity    float32 `json:"humidity"`
}

func NewFromWeather(ctx context.Context, w *Weather) (*Event, error) {
	payload, err := json.Marshal(w)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal weather: %w", err)
	}
	return &Event{
		ID:        uuid.NewString(),
		EventType: TerraceWeatherEventType,
		EventAt:   time.Now(),
		Payload:   payload,
		Trace:     buildTracingMap(ctx),
	}, nil
}

func NewFromExecutionLog(ctx context.Context, lo *Log) (*Event, error) {
	payload, err := json.Marshal(lo)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal execution log: %w", err)
	}
	return &Event{
		ID:        fmt.Sprintf("%s-%v", lo.ZoneName, lo.ExecutedAt.Unix()),
		EventType: ExecutionLogsEventType,
		EventAt:   time.Now(),
		Payload:   payload,
		Trace:     buildTracingMap(ctx),
	}, nil
}

func buildTracingMap(ctx context.Context) map[string]string {
	carrier := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(carrier))
	return carrier
}

type EventsRepository struct {
	eventsDir string
	tracer    trace.Tracer
}

func (e EventsRepository) Save(ctx context.Context, ev *Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := e.tracer.Start(ctx, "EventsRepository.Save")
		defer span.End()
		if err := os.MkdirAll(e.eventsDir, 0o755); err != nil {
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

func (e EventsRepository) Remove(ctx context.Context, ev *Event) error {
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

func (e EventsRepository) FindAll(ctx context.Context) ([]Event, error) {
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
		events := make([]Event, 0, len(files))
		for _, f := range files {
			if !isEventFile(f) {
				continue
			}
			path := filepath.Join(e.eventsDir, f.Name())
			var ev Event
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
	deadLetterDir := filepath.Join(e.eventsDir, DeadLetterDirectory)
	if err := os.MkdirAll(deadLetterDir, 0o755); err != nil {
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

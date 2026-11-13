package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const (
	ExecutionLogsEventType  = "execution_logs"
	TerraceWeatherEventType = "terrace_weather"
)

type Event struct {
	ID        string            `json:"id"`
	EventType string            `json:"event_type"`
	EventAt   time.Time         `json:"event_at"`
	Payload   []byte            `json:"payload"`
	Trace     map[string]string `json:"trace"`
}

type ExecutionLog struct {
	Seconds    int       `json:"seconds"`
	ZoneName   string    `json:"zone_name"`
	ExecutedAt time.Time `json:"executed_at"`
}

type Weather struct {
	Temperature float32 `json:"temperature"`
	IsRaining   bool    `json:"is_raining"`
	Humidity    float32 `json:"humidity"`
}

func NewFromWeather(ctx context.Context, w Weather, now time.Time) (*Event, error) {
	payload, err := json.Marshal(w)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal weather: %w", err)
	}
	return &Event{
		ID:        uuid.NewString(),
		EventType: TerraceWeatherEventType,
		EventAt:   now,
		Payload:   payload,
		Trace:     buildTracingMap(ctx),
	}, nil
}

func NewFromExecutionLog(ctx context.Context, lo ExecutionLog, now time.Time) (*Event, error) {
	payload, err := json.Marshal(lo)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal execution log: %w", err)
	}
	return &Event{
		ID:        fmt.Sprintf("%s-%v", lo.ZoneName, lo.ExecutedAt.Unix()),
		EventType: ExecutionLogsEventType,
		EventAt:   now,
		Payload:   payload,
		Trace:     buildTracingMap(ctx),
	}, nil
}

func buildTracingMap(ctx context.Context) map[string]string {
	carrier := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(carrier))
	return carrier
}

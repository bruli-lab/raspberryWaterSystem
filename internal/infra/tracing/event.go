package tracing

import (
	"github.com/bruli-lab/go-core/event"
	"go.opentelemetry.io/otel/trace"
)

type Event struct {
	SpanContext trace.SpanContext
	Event       event.Event
}

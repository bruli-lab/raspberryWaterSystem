package app

import (
	"github.com/bruli-lab/go-core/event"
	"go.opentelemetry.io/otel/trace"
)

type TracedEvent struct {
	SpanContext trace.SpanContext
	Event       event.Event
}

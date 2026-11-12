package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/infra/tracing"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/stretchr/testify/require"
)

func TestNewEventMiddleware(t *testing.T) {
	t.Run(`Given an event command handler middleware,
	when is built with an event channel,
	and command handler return events,
	then all events are write in channel`, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		errTest := errors.New("")
		ev := Event{
			ID:        uuid.New(),
			Name:      "eventito 1",
			At:        time.Now(),
			AggRootID: "dkdkdkd",
		}
		event1 := tracing.Event{
			SpanContext: trace.SpanContext{},
			Event:       ev,
		}
		eventCh := make(chan tracing.Event)
		eventMdw := app.NewEventMiddleware(eventCh, tracer())
		handler := eventMdw(commandHandler{
			events: []event.Event{
				ev,
			},
			err: errTest,
		})
		go func() {
			for {
				select {
				case <-ctx.Done():
					cancel()
					return
				case received := <-eventCh:
					require.Equal(t, event1, received)
				}
			}
		}()
		evnts, err := handler.Handle(ctx, command{})
		require.Nil(t, evnts)
		require.Equal(t, errTest, err)
	})
}

type Event struct {
	ID        uuid.UUID
	Name      string
	At        time.Time
	AggRootID string
}

func (e Event) EventID() uuid.UUID {
	return e.ID
}

func (e Event) EventName() string {
	return e.Name
}

func (e Event) EventAt() time.Time {
	return e.At
}

func (e Event) AggregateRootID() string {
	return e.AggRootID
}

type command struct {
	name string
}

func (c command) Name() string {
	return c.name
}

type commandHandler struct {
	events []event.Event
	err    error
}

func (c commandHandler) Handle(_ context.Context, _ cqs.Command) ([]event.Event, error) {
	return c.events, c.err
}

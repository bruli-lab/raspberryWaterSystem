package nats

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const StreamName = "WATER_SYSTEM"

type NotConnectedError struct{}

func (NotConnectedError) Error() string {
	return "nats server is not connected"
}

type Publisher struct {
	nc          *nats.Conn
	js          nats.JetStreamContext
	tracer      trace.Tracer
	subjects    []string
	m           sync.Mutex
	streamReady bool
}

func (p *Publisher) IsConnected() bool {
	return p.nc.IsConnected()
}

func (p *Publisher) PublishEvent(ctx context.Context, id, subject string, payload []byte) error {
	errs := p.PublishEvents(ctx, []outbox.Event{{ID: id, EventType: subject, Payload: payload}})
	return errs[0]
}

func (p *Publisher) PublishEvents(ctx context.Context, events []outbox.Event) []error {
	errs := make([]error, len(events))
	futures := make([]nats.PubAckFuture, len(events))
	spans := make([]trace.Span, len(events))
	for i := range events {
		eventCtx, span := p.tracer.Start(ctx, "publish-nats-event")
		spans[i] = span
		future, err := p.submitEvent(eventCtx, &events[i])
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			errs[i] = err
			continue
		}
		futures[i] = future
	}
	for i, future := range futures {
		if future == nil {
			spans[i].End()
			continue
		}
		if errs[i] = waitForAck(ctx, future); errs[i] != nil {
			spans[i].RecordError(errs[i])
			spans[i].SetStatus(codes.Error, errs[i].Error())
		} else {
			spans[i].SetStatus(codes.Ok, "event published")
		}
		spans[i].End()
	}
	return errs
}

func (p *Publisher) submitEvent(ctx context.Context, ev *outbox.Event) (nats.PubAckFuture, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !p.IsConnected() {
		return nil, NotConnectedError{}
	}
	if err := p.ensureStream(); err != nil {
		return nil, err
	}
	msg := buildEventMsg(ctx, ev)
	future, err := p.js.PublishMsgAsync(&msg)
	if err != nil {
		return nil, fmt.Errorf("failed to publish event: %w", err)
	}
	return future, nil
}

func waitForAck(ctx context.Context, future nats.PubAckFuture) error {
	select {
	case ack := <-future.Ok():
		if ack == nil {
			return errors.New("failed to publish event: nil puback")
		}
		return nil
	case err := <-future.Err():
		return fmt.Errorf("failed to publish event: %w", err)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func buildEventMsg(ctx context.Context, ev *outbox.Event) nats.Msg {
	msg := nats.Msg{
		Subject: ev.EventType,
		Data:    ev.Payload,
		Header: nats.Header{
			"Nats-Msg-Id": []string{ev.ID},
		},
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(msg.Header))
	return msg
}

func (p *Publisher) ensureStream() error {
	p.m.Lock()
	defer p.m.Unlock()
	if p.streamReady {
		return nil
	}
	if _, err := p.js.StreamInfo(StreamName); err != nil {
		if !errors.Is(err, nats.ErrStreamNotFound) {
			return fmt.Errorf("failed to get stream info: %w", err)
		}
		if _, err = p.js.AddStream(&nats.StreamConfig{
			Name:       StreamName,
			Subjects:   p.subjects,
			Storage:    nats.FileStorage,
			Retention:  nats.WorkQueuePolicy,
			MaxAge:     7 * 24 * time.Hour,
			Replicas:   1,
			Duplicates: 10 * time.Minute,
		}); err != nil {
			return fmt.Errorf("failed to create stream: %w", err)
		}
	}
	p.streamReady = true
	return nil
}

func (p *Publisher) Close() {
	if p.nc != nil {
		p.nc.Close()
	}
}

func NewPublisher(natsURL string, subjects []string, tracer trace.Tracer) (*Publisher, error) {
	nc, err := nats.Connect(
		natsURL,
		nats.Name("water-system"),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create jetstream context: %w", err)
	}

	return &Publisher{
		nc:       nc,
		js:       js,
		tracer:   tracer,
		subjects: subjects,
	}, nil
}

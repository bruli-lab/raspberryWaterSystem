package ntfy

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Publisher struct {
	user      string
	password  string
	serverURL *url.URL
	topic     string
	tracer    trace.Tracer
	client    *http.Client
}

func (p *Publisher) Publish(ctx context.Context, message string) error {
	ctx, span := p.tracer.Start(ctx, "Publisher.Publish")
	defer span.End()
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/%s", p.serverURL.String(), p.topic),
		bytes.NewBufferString(message),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(p.user, p.password)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Title", "WaterSystem")
	req.Header.Set("Priority", "default")

	resp, err := p.client.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 300 {
		err := fmt.Errorf("ntfy returned status %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "message published")
	return nil
}

func NewPublisher(user, password, baseUrl, topic string, tracer trace.Tracer) (*Publisher, error) {
	serverUrl, err := url.Parse(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse publisher url: %w", err)
	}
	pu := Publisher{user: user, password: password, serverURL: serverUrl, topic: topic, tracer: tracer, client: &http.Client{Timeout: 10 * time.Second}}
	return &pu, nil
}

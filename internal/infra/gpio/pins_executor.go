package gpio

import (
	"context"
	"fmt"
	"time"

	"github.com/warthog618/go-gpiocdev"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const gpioChip = "gpiochip0"

type PinsExecutor struct {
	relays map[string]int
	tracer trace.Tracer
}

func (p *PinsExecutor) Execute(ctx context.Context, seconds uint, pins []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	ctx, span := p.tracer.Start(ctx, "PinsExecutor.Execute")
	defer span.End()

	activatedPins := make([]*gpiocdev.Line, 0, len(pins))

	defer func() {
		for _, line := range activatedPins {
			_ = line.SetValue(1)
			_ = line.Close()
		}
	}()

	for _, pinNumber := range pins {
		line, err := p.activatePin(pinNumber)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		activatedPins = append(activatedPins, line)
	}

	timer := time.NewTimer(time.Duration(seconds) * time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		span.RecordError(ctx.Err())
		span.SetStatus(codes.Error, ctx.Err().Error())
		return ctx.Err()

	case <-timer.C:
	}

	span.SetStatus(codes.Ok, "pins executed")
	return nil
}

func (p *PinsExecutor) activatePin(pinNumber string) (*gpiocdev.Line, error) {
	lineNumber, ok := p.relays[pinNumber]
	if !ok {
		return nil, InvalidPinToExecuteError{pinNumber: pinNumber}
	}

	line, err := gpiocdev.RequestLine(
		gpioChip,
		lineNumber,
		gpiocdev.AsOutput(1),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to request GPIO %s: %w", pinNumber, err)
	}

	if err := line.SetValue(0); err != nil {
		_ = line.Close()
		return nil, fmt.Errorf("failed to activate GPIO %s: %w", pinNumber, err)
	}

	return line, nil
}

func NewPinsExecutor(tracer trace.Tracer) *PinsExecutor {
	return &PinsExecutor{
		relays: relays,
		tracer: tracer,
	}
}

type InvalidPinToExecuteError struct {
	pinNumber string
}

func (i InvalidPinToExecuteError) Error() string {
	return fmt.Sprintf("invalid pin %q to execute", i.pinNumber)
}

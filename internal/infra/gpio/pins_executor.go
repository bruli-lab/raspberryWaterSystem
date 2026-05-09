package gpio

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"periph.io/x/conn/v3/gpio"
)

type PinsExecutor struct {
	relays map[string]gpio.PinIO
	tracer trace.Tracer
}

func NewPinsExecutor(tracer trace.Tracer) *PinsExecutor {
	return &PinsExecutor{relays: relays, tracer: tracer}
}

func (p *PinsExecutor) Execute(ctx context.Context, seconds uint, pins []string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := p.tracer.Start(ctx, "PinsExecutor.Execute")
		defer span.End()
		activatedPins := make([]gpio.PinIO, len(pins))
		for i, piNumber := range pins {
			activatePin, err := p.activatePin(piNumber)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return err
			}
			activatedPins[i] = activatePin
		}
		time.Sleep(time.Duration(seconds) * time.Second)
		for _, act := range activatedPins {
			if err := p.deActivatePin(act); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return err
			}
		}
		span.SetStatus(codes.Ok, "pins executed")
		return nil
	}
}

func (p *PinsExecutor) activatePin(piNumber string) (gpio.PinIO, error) {
	pi, ok := p.relays[piNumber]
	if !ok {
		return nil, InvalidPinToExecuteError{pinNumber: piNumber}
	}
	if err := pi.Out(gpio.Low); err != nil {
		return nil, err
	}
	return pi, nil
}

func (p *PinsExecutor) deActivatePin(pi gpio.PinIO) error {
	return pi.Out(gpio.High)
}

type InvalidPinToExecuteError struct {
	pinNumber string
}

func (i InvalidPinToExecuteError) Error() string {
	return fmt.Sprintf("invalid pin %q to execute", i.pinNumber)
}

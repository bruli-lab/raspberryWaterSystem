package gpio

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/warthog618/go-gpiocdev"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	gpioChip = "gpiochip0"
	relayOn  = 0
	relayOff = 1
)

type PinsExecutor struct {
	relays  map[string]int
	tracer  trace.Tracer
	m       sync.Mutex
	active  map[string]*gpiocdev.Line
	stopped bool
}

func (p *PinsExecutor) Execute(ctx context.Context, seconds uint, pins []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	ctx, span := p.tracer.Start(ctx, "PinsExecutor.Execute")
	defer span.End()

	activatedPins := make([]string, 0, len(pins))
	defer func() {
		p.deactivatePins(activatedPins)
	}()

	for _, pinNumber := range pins {
		if err := p.activatePin(pinNumber); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		activatedPins = append(activatedPins, pinNumber)
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

func (p *PinsExecutor) TurnOffAll(ctx context.Context) error {
	_, span := p.tracer.Start(ctx, "PinsExecutor.TurnOffAll")
	defer span.End()
	p.m.Lock()
	defer p.m.Unlock()
	if err := p.turnOffAll(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "relays turned off")
	return nil
}

func (p *PinsExecutor) Stop(ctx context.Context) error {
	_, span := p.tracer.Start(ctx, "PinsExecutor.Stop")
	defer span.End()
	p.m.Lock()
	defer p.m.Unlock()
	p.stopped = true
	if err := p.turnOffAll(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "relays stopped")
	return nil
}

func (p *PinsExecutor) turnOffAll() error {
	var errs []error
	for pinNumber, lineNumber := range p.relays {
		if line, ok := p.active[pinNumber]; ok {
			if err := line.SetValue(relayOff); err != nil {
				errs = append(errs, fmt.Errorf("failed to turn off GPIO %s: %w", pinNumber, err))
			}
			continue
		}
		line, err := gpiocdev.RequestLine(gpioChip, lineNumber, gpiocdev.AsOutput(relayOff))
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to request GPIO %s: %w", pinNumber, err))
			continue
		}
		if err = line.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to release GPIO %s: %w", pinNumber, err))
		}
	}
	return errors.Join(errs...)
}

func (p *PinsExecutor) activatePin(pinNumber string) error {
	lineNumber, ok := p.relays[pinNumber]
	if !ok {
		return InvalidPinToExecuteError{pinNumber: pinNumber}
	}
	p.m.Lock()
	defer p.m.Unlock()
	if p.stopped {
		return RelaysStoppedError{}
	}
	if _, busy := p.active[pinNumber]; busy {
		return PinAlreadyActiveError{pinNumber: pinNumber}
	}

	line, err := gpiocdev.RequestLine(
		gpioChip,
		lineNumber,
		gpiocdev.AsOutput(relayOff),
	)
	if err != nil {
		return fmt.Errorf("failed to request GPIO %s: %w", pinNumber, err)
	}

	if err := line.SetValue(relayOn); err != nil {
		_ = line.SetValue(relayOff)
		_ = line.Close()
		return fmt.Errorf("failed to activate GPIO %s: %w", pinNumber, err)
	}
	p.active[pinNumber] = line
	return nil
}

func (p *PinsExecutor) deactivatePins(pins []string) {
	p.m.Lock()
	defer p.m.Unlock()
	for _, pinNumber := range pins {
		line, ok := p.active[pinNumber]
		if !ok {
			continue
		}
		_ = line.SetValue(relayOff)
		_ = line.Close()
		delete(p.active, pinNumber)
	}
}

func NewPinsExecutor(tracer trace.Tracer) *PinsExecutor {
	return &PinsExecutor{
		relays: relays,
		tracer: tracer,
		active: make(map[string]*gpiocdev.Line),
	}
}

type InvalidPinToExecuteError struct {
	pinNumber string
}

func (i InvalidPinToExecuteError) Error() string {
	return fmt.Sprintf("invalid pin %q to execute", i.pinNumber)
}

type PinAlreadyActiveError struct {
	pinNumber string
}

func (p PinAlreadyActiveError) Error() string {
	return fmt.Sprintf("pin %q is already active", p.pinNumber)
}

type RelaysStoppedError struct{}

func (RelaysStoppedError) Error() string {
	return "relays are stopped"
}

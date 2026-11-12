package fake

import (
	"context"
	"time"
)

type PinsExecutor struct{}

func NewPinsExecutor() PinsExecutor {
	return PinsExecutor{}
}

func (p PinsExecutor) Execute(ctx context.Context, seconds uint, _ []string) error {
	timer := time.NewTimer(time.Duration(seconds) * time.Second)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (p PinsExecutor) TurnOffAll(_ context.Context) error {
	return nil
}

func (p PinsExecutor) Stop(_ context.Context) error {
	return nil
}

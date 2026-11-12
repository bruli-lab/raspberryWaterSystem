package app

import "context"

//go:generate go tool moq -out pin_executor_mock_test.go --pkg app_test . PinExecutor
type PinExecutor interface {
	Execute(ctx context.Context, seconds uint, pins []string) error
}

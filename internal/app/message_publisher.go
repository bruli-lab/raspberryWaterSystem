package app

import (
	"context"
)

//go:generate go tool moq -out message_publisher_mock_test.go --pkg app_test . MessagePublisher

type MessagePublisher interface {
	Publish(ctx context.Context, message string) error
}

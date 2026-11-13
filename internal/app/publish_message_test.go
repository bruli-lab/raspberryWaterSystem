package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/stretchr/testify/require"
)

func TestPublishExecutionLogHandle(t *testing.T) {
	errTest := errors.New("")
	cmd := app.PublishMessageCmd{
		Message: "published",
	}
	tests := []struct {
		name                    string
		publishErr, expectedErr error
		cmd                     cqs.Command
	}{
		{
			name:        "with an invalid command, then it returns an invalid command error",
			cmd:         invalidCommand{},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:        "and publish returns an error, then it returns same error",
			cmd:         cmd,
			publishErr:  errTest,
			expectedErr: errTest,
		},
		{
			name: "then it returns nil",
			cmd:  cmd,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a PublishMessage command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			elp := &MessagePublisherMock{
				PublishFunc: func(_ context.Context, _ string) error {
					return tt.publishErr
				},
			}
			handler := app.NewPublishMessage(elp, tracer())
			events, err := handler.Handle(context.Background(), tt.cmd)
			if err != nil {
				require.IsType(t, tt.expectedErr, err)
				return
			}
			require.Equal(t, tt.expectedErr, err)
			require.Nil(t, events)
		})
	}
}

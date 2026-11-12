package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestExecutePinsHandle(t *testing.T) {
	errTest := errors.New("")
	cmd := app.ExecutePinsCmd{Seconds: 10, Pins: []string{"18"}}
	cleanCycleCmd := app.ExecutePinsCmd{Seconds: 10, Pins: []string{"18"}, CleanCycle: true}
	idle := fixtures.StatusBuilder{Active: true}.Build()
	cleaning := fixtures.StatusBuilder{Active: true, Clean: true}.Build()
	tests := []struct {
		name                          string
		execErr, findErr, expectedErr error
		st                            status.Status
		cmd                           cqs.Command
		expectedExecutions            int
	}{
		{
			name:        "with an invalid command, then it returns an invalid command error",
			cmd:         invalidCommand{},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:        "and find status returns an error, then it returns same error without executing pins",
			cmd:         cmd,
			findErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and a clean cycle is in progress, then it discards the execution with a clean in progress error",
			cmd:         cmd,
			st:          cleaning,
			expectedErr: app.CleanInProgressError{},
		},
		{
			name:               "from the clean cycle while it is in progress, then it executes pins",
			cmd:                cleanCycleCmd,
			st:                 cleaning,
			expectedExecutions: 1,
		},
		{
			name:               "and executor returns an error, then it returns same error",
			cmd:                cmd,
			st:                 idle,
			execErr:            errTest,
			expectedErr:        errTest,
			expectedExecutions: 1,
		},
		{
			name:               "and executor returns nil, then it returns nil events",
			cmd:                cmd,
			st:                 idle,
			expectedExecutions: 1,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an ExecutePins command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			pe := &PinExecutorMock{
				ExecuteFunc: func(_ context.Context, _ uint, _ []string) error {
					return tt.execErr
				},
			}
			sr := &StatusRepositoryMock{
				FindFunc: func(_ context.Context) (status.Status, error) {
					return tt.st, tt.findErr
				},
			}
			handler := app.NewExecutePins(pe, sr, tracer())
			events, err := handler.Handle(context.Background(), tt.cmd)
			require.Len(t, pe.ExecuteCalls(), tt.expectedExecutions)
			if tt.expectedErr != nil {
				require.ErrorAs(t, err, &tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Nil(t, events)
		})
	}
}

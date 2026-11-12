package app_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/stretchr/testify/require"
)

func TestSaveExecutionLogHandle(t *testing.T) {
	errTest := errors.New("")
	cmd := app.SaveExecutionLogCmd{
		ZoneName:   "zone new",
		Seconds:    program.Seconds(20 * time.Second),
		ExecutedAt: time.Now(),
	}
	logs := make([]program.ExecutionLog, 25)
	for i := 0; 25 > i; i++ {
		logs[i] = fixtures.ExecutionLogBuilder{
			ZoneName: new(fmt.Sprintf("zone %v", i)),
		}.Build()
	}
	tests := []struct {
		name string
		cmd  cqs.Command
		expectedErr, findErr,
		saveErr error
		logs []program.ExecutionLog
	}{
		{
			name:        "with an invalid command, then it returns an invalid command error",
			cmd:         invalidCommand{},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:        "and find all returns an error, then it returns same error",
			cmd:         cmd,
			findErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and new execution returns an error, then it returns a save execution log error",
			cmd:         app.SaveExecutionLogCmd{},
			logs:        logs,
			expectedErr: app.SaveExecutionLogError{},
		},
		{
			name:        "and save returns an error, then it returns same error",
			cmd:         cmd,
			logs:        logs,
			saveErr:     errTest,
			expectedErr: errTest,
		},
		{
			name: "then it returns nil",
			cmd:  cmd,
			logs: logs,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a SaveExecutionLog command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			elr := &ExecutionLogRepositoryMock{
				FindAllFunc: func(_ context.Context) ([]program.ExecutionLog, error) {
					return tt.logs, tt.findErr
				},
				SaveFunc: func(_ context.Context, _ []program.ExecutionLog) error {
					return tt.saveErr
				},
			}
			handler := app.NewSaveExecutionLog(elr, tracer())
			events, err := handler.Handle(context.Background(), tt.cmd)
			if err != nil {
				require.ErrorAs(t, err, &tt.expectedErr)
				return
			}
			require.Equal(t, tt.expectedErr, err)
			require.Nil(t, events)
		})
	}
}

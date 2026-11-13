package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/stretchr/testify/require"
)

func TestRemoveWeekly_Handle(t *testing.T) {
	errTest := errors.New("")
	type args struct {
		ctx context.Context
		cmd cqs.Command
	}
	defaultArgs := args{
		ctx: t.Context(),
		cmd: app.RemoveWeeklyProgramCommand{},
	}
	tests := []struct {
		name string
		args args
		findErr, removeErr,
		expectedErr error
	}{
		{
			name: "with an invalid command, then it returns an invalid command error",
			args: args{
				ctx: t.Context(),
				cmd: invalidCommand{},
			},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:        "and find program returns an error, then it returns same error",
			args:        defaultArgs,
			findErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and remove program returns an error, then it returns same error",
			args:        defaultArgs,
			removeErr:   errTest,
			expectedErr: errTest,
		},
		{
			name: "and remove program returns nil, then it returns nil",
			args: defaultArgs,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a RemoveWeeklyProgram command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &WeeklyProgramRepositoryMock{
				FindByDayFunc: func(_ context.Context, _ *program.WeekDay) (*program.Weekly, error) {
					return nil, tt.findErr
				},
				RemoveFunc: func(_ context.Context, _ *program.WeekDay) error {
					return tt.removeErr
				},
			}
			handler := app.NewRemoveWeeklyProgram(repo, tracer())
			_, err := handler.Handle(tt.args.ctx, tt.args.cmd)
			if err != nil {
				require.IsType(t, tt.expectedErr, err)
			}
		})
	}
}

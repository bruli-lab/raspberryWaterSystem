package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/stretchr/testify/require"
)

func TestRemoveDailyProgram_Handle(t *testing.T) {
	errTest := errors.New("")
	hour, err := program.ParseHour("15:00")
	require.NoError(t, err)
	ctx := context.Background()
	type args struct {
		ctx context.Context
		cmd cqs.Command
	}
	defaultArgs := args{
		ctx: ctx,
		cmd: app.RemoveDailyProgramCommand{
			Hour: &hour,
		},
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
				ctx: context.Background(),
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
			name:        "and save program returns an error, then it returns same error",
			args:        defaultArgs,
			removeErr:   errTest,
			expectedErr: errTest,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a RemoveDailyProgram,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &ProgramRepositoryMock{
				FindByHourFunc: func(_ context.Context, _ *program.Hour) (*program.FertilizerProgram, error) {
					return nil, tt.findErr
				},
				RemoveFunc: func(_ context.Context, _ *program.Hour) error {
					return tt.removeErr
				},
			}
			handler := app.NewRemoveDailyProgram(repo, tracer())
			_, err := handler.Handle(tt.args.ctx, tt.args.cmd)
			if err != nil {
				require.ErrorAs(t, err, &tt.expectedErr)
			}
		})
	}
}

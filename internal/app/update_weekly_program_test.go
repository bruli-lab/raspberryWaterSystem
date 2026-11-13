package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestUpdateWeeklyProgram_Handle(t *testing.T) {
	errTest := errors.New("")
	type args struct {
		ctx context.Context
		cmd cqs.Command
	}
	weekly := fixtures.WeeklyBuilder{}.Build()
	defaultArgs := args{
		ctx: t.Context(),
		cmd: app.UpdateWeeklyProgramCommand{
			WeekDay:  weekly.WeekDay(),
			Programs: weekly.Programs(),
		},
	}
	tests := []struct {
		name string
		args args
		expectedErr, findErr,
		saveErr error
		weekly program.Weekly
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
			name:        "and find weekly program returns an error, then it returns same error",
			args:        defaultArgs,
			findErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and save method returns an error, then it returns same error",
			args:        defaultArgs,
			weekly:      weekly,
			saveErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:   "and save method returns nil, then it returns nil",
			args:   defaultArgs,
			weekly: weekly,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an UpdateWeeklyProgram command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &WeeklyProgramRepositoryMock{
				FindByDayFunc: func(_ context.Context, _ *program.WeekDay) (*program.Weekly, error) {
					return &tt.weekly, tt.findErr
				},
				SaveFunc: func(_ context.Context, _ *program.Weekly) error {
					return tt.saveErr
				},
			}
			handler := app.NewUpdateWeeklyProgram(repo, tracer())

			_, err := handler.Handle(tt.args.ctx, tt.args.cmd)
			if err != nil {
				require.IsType(t, tt.expectedErr, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

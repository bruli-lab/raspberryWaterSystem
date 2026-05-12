package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestCreateProgram_Handle(t *testing.T) {
	ctx := context.Background()
	errTest := errors.New("")
	type args struct {
		ctx context.Context
		cmd cqs.Command
	}
	prog := fixtures.FertilizerProgramBuilder{}.Build()
	defaultArgs := args{
		ctx: ctx,
		cmd: app.CreateDailyProgramCommand{
			Program: &prog,
		},
	}
	tests := []struct {
		name string
		args args
		expectedErr, findErr,
		saveErr, zoneErr error
	}{
		{
			name: "with an invalid command, then it returns an invalid command error",
			args: args{
				ctx: ctx,
				cmd: invalidCommand{},
			},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:        "and find hour returns a nil error, then it returns a create program error",
			args:        defaultArgs,
			expectedErr: app.CreateProgramError{},
		},
		{
			name:        "and find hour returns an error, then it returns same error",
			args:        defaultArgs,
			findErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and find zone returns a not found error, then it returns a create program error",
			args:        defaultArgs,
			findErr:     errs.NotFoundError{},
			zoneErr:     errs.NotFoundError{},
			expectedErr: app.CreateProgramError{},
		},
		{
			name:        "and find zone returns an error, then it returns same error",
			args:        defaultArgs,
			findErr:     errs.NotFoundError{},
			zoneErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and save method returns an error, then it returns same error",
			args:        defaultArgs,
			findErr:     errs.NotFoundError{},
			saveErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:    "and save method returns nil, then it returns nil",
			args:    defaultArgs,
			findErr: errs.NotFoundError{},
		},
	}
	for _, tt := range tests {
		t.Run(`Given CreateDailyProgram command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			programRepo := &ProgramRepositoryMock{
				FindByHourFunc: func(_ context.Context, _ *program.Hour) (*program.FertilizerProgram, error) {
					return nil, tt.findErr
				},
				SaveFunc: func(_ context.Context, _ *program.FertilizerProgram) error {
					return tt.saveErr
				},
			}
			zonesRepo := &ZoneRepositoryMock{
				FindByIDFunc: func(_ context.Context, _ string) (*zone.Zone, error) {
					return nil, tt.zoneErr
				},
			}

			handler := app.NewCreateDailyProgram(programRepo, zonesRepo, tracer())
			_, err := handler.Handle(tt.args.ctx, tt.args.cmd)
			if err != nil {
				require.IsType(t, tt.expectedErr, err)
			}
		})
	}
}

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

func TestUpdateDailyProgram_Handle(t *testing.T) {
	ctx := context.Background()
	errTest := errors.New("")
	type args struct {
		ctx context.Context
		cmd cqs.Command
	}
	prog := fixtures.ProgramBuilder{}.Build()
	defaultArgs := args{
		ctx: ctx,
		cmd: app.UpdateDailyProgramCommand{
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
			name:        "and find hour returns a not found error, then it returns same error",
			args:        defaultArgs,
			findErr:     errs.NotFoundError{},
			expectedErr: errs.NotFoundError{},
		},
		{
			name:        "and find hour returns an error, then it returns same error",
			args:        defaultArgs,
			findErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and find zone returns a not found error, then it returns an update program error",
			args:        defaultArgs,
			zoneErr:     errs.NotFoundError{},
			expectedErr: app.UpdateProgramError{},
		},
		{
			name:        "and find zone returns an error, then it returns same error",
			args:        defaultArgs,
			zoneErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and save method returns an error, then it returns same error",
			args:        defaultArgs,
			saveErr:     errTest,
			expectedErr: errTest,
		},
		{
			name: "and save method returns nil, then it returns nil",
			args: defaultArgs,
		},
	}
	for _, tt := range tests {
		t.Run(`Given UpdateDailyProgram command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			programRepo := &ProgramRepositoryMock{
				FindByHourFunc: func(_ context.Context, _ *program.Hour) (*program.Program, error) {
					return nil, tt.findErr
				},
				SaveFunc: func(_ context.Context, _ *program.Program) error {
					return tt.saveErr
				},
			}
			zonesRepo := &ZoneRepositoryMock{
				FindByIDFunc: func(_ context.Context, _ string) (*zone.Zone, error) {
					return nil, tt.zoneErr
				},
			}

			handler := app.NewUpdateDailyProgram(programRepo, zonesRepo, tracer())
			_, err := handler.Handle(tt.args.ctx, tt.args.cmd)
			if err != nil {
				require.IsType(t, tt.expectedErr, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUpdateOddAndEvenProgram_Handle(t *testing.T) {
	prog := fixtures.ProgramBuilder{}.Build()
	tests := []struct {
		name        string
		cmd         cqs.Command
		newHandler  func(programRepo app.ProgramRepository, zonesRepo app.ZoneRepository) cqs.CommandHandler
		expectedErr any
	}{
		{
			name: "UpdateOddProgram with an invalid command, then it returns an invalid command error",
			cmd:  invalidCommand{},
			newHandler: func(p app.ProgramRepository, z app.ZoneRepository) cqs.CommandHandler {
				return app.NewUpdateOddProgram(p, z, tracer())
			},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name: "UpdateOddProgram with a valid command, then it updates the program",
			cmd:  app.UpdateOddProgramCommand{Program: &prog},
			newHandler: func(p app.ProgramRepository, z app.ZoneRepository) cqs.CommandHandler {
				return app.NewUpdateOddProgram(p, z, tracer())
			},
		},
		{
			name: "UpdateEvenProgram with an invalid command, then it returns an invalid command error",
			cmd:  invalidCommand{},
			newHandler: func(p app.ProgramRepository, z app.ZoneRepository) cqs.CommandHandler {
				return app.NewUpdateEvenProgram(p, z, tracer())
			},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name: "UpdateEvenProgram with a valid command, then it updates the program",
			cmd:  app.UpdateEvenProgramCommand{Program: &prog},
			newHandler: func(p app.ProgramRepository, z app.ZoneRepository) cqs.CommandHandler {
				return app.NewUpdateEvenProgram(p, z, tracer())
			},
		},
	}
	for _, tt := range tests {
		t.Run(`Given an update program command handler, when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			programRepo := &ProgramRepositoryMock{
				FindByHourFunc: func(_ context.Context, _ *program.Hour) (*program.Program, error) {
					return nil, nil
				},
				SaveFunc: func(_ context.Context, _ *program.Program) error {
					return nil
				},
			}
			zonesRepo := &ZoneRepositoryMock{
				FindByIDFunc: func(_ context.Context, _ string) (*zone.Zone, error) {
					return nil, nil
				},
			}
			handler := tt.newHandler(programRepo, zonesRepo)

			_, err := handler.Handle(t.Context(), tt.cmd)

			if tt.expectedErr != nil {
				require.IsType(t, tt.expectedErr, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

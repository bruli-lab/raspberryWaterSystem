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

func updateProgramInput() app.UpdateProgramInput {
	return app.UpdateProgramInput{
		Hour:       "15:10",
		Status:     "disabled",
		Fertilizer: true,
		Executions: []app.ProgramExecutionInput{{Seconds: 30, Zones: []string{"2"}}},
	}
}

func TestUpdateDailyProgram_Handle(t *testing.T) {
	errTest := errors.New("")
	withInput := func(change func(in *app.UpdateProgramInput)) cqs.Command {
		in := updateProgramInput()
		change(&in)
		return app.UpdateDailyProgramCommand{UpdateProgramInput: in}
	}
	defaultCmd := app.UpdateDailyProgramCommand{UpdateProgramInput: updateProgramInput()}
	tests := []struct {
		name string
		cmd  cqs.Command
		expectedErr, findErr,
		saveErr, zoneErr error
	}{
		{
			name:        "with an invalid command, then it returns an invalid command error",
			cmd:         invalidCommand{},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:        "with an invalid hour, then it returns an update program error",
			cmd:         withInput(func(in *app.UpdateProgramInput) { in.Hour = "invalid" }),
			expectedErr: app.UpdateProgramError{},
		},
		{
			name:        "with invalid seconds, then it returns an update program error",
			cmd:         withInput(func(in *app.UpdateProgramInput) { in.Executions[0].Seconds = -10 }),
			expectedErr: app.UpdateProgramError{},
		},
		{
			name:        "with an invalid status, then it returns an update program error",
			cmd:         withInput(func(in *app.UpdateProgramInput) { in.Status = "paused" }),
			expectedErr: app.UpdateProgramError{},
		},
		{
			name:        "without executions, then it returns an update program error",
			cmd:         withInput(func(in *app.UpdateProgramInput) { in.Executions = nil }),
			expectedErr: app.UpdateProgramError{},
		},
		{
			name:        "and find hour returns a not found error, then it returns same error",
			cmd:         defaultCmd,
			findErr:     errs.NotFoundError{},
			expectedErr: errs.NotFoundError{},
		},
		{
			name:        "and find hour returns an error, then it returns same error",
			cmd:         defaultCmd,
			findErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and find zone returns a not found error, then it returns an update program error",
			cmd:         defaultCmd,
			zoneErr:     errs.NotFoundError{},
			expectedErr: app.UpdateProgramError{},
		},
		{
			name:        "and find zone returns an error, then it returns same error",
			cmd:         defaultCmd,
			zoneErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and save method returns an error, then it returns same error",
			cmd:         defaultCmd,
			saveErr:     errTest,
			expectedErr: errTest,
		},
		{
			name: "and save method returns nil, then it saves the found program updated",
			cmd:  defaultCmd,
		},
	}
	for _, tt := range tests {
		t.Run(`Given UpdateDailyProgram command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			hour, err := program.ParseHour("15:10")
			require.NoError(t, err)
			current := fixtures.ProgramBuilder{Hour: &hour}.Build()
			programRepo := &ProgramRepositoryMock{
				FindByHourFunc: func(_ context.Context, _ *program.Hour) (*program.Program, error) {
					if tt.findErr != nil {
						return nil, tt.findErr
					}
					return &current, nil
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
			_, err = handler.Handle(t.Context(), tt.cmd)
			if tt.expectedErr != nil {
				require.IsType(t, tt.expectedErr, err)
				if tt.saveErr == nil {
					require.Empty(t, programRepo.SaveCalls())
				}
				return
			}
			require.NoError(t, err)
			require.Len(t, programRepo.SaveCalls(), 1)
			saved := programRepo.SaveCalls()[0].Programs
			require.Same(t, &current, saved)
			require.Equal(t, hour, saved.Hour())
			require.True(t, saved.IsFertilizer())
			require.Equal(t, program.DisabledStatus, saved.Status())
			require.Equal(t, []string{"2"}, saved.Executions()[0].Zones())
			require.Equal(t, 30, saved.Executions()[0].Seconds().Int())
		})
	}
}

func TestUpdateProgramWithoutStatus(t *testing.T) {
	t.Run(`Given UpdateDailyProgram command handler and a disabled program,
	when Handle method is called without status,
	then it saves the program enabled`, func(t *testing.T) {
		t.Parallel()
		disabled := program.DisabledStatus
		current := fixtures.ProgramBuilder{Status: &disabled}.Build()
		programRepo := &ProgramRepositoryMock{
			FindByHourFunc: func(_ context.Context, _ *program.Hour) (*program.Program, error) {
				return &current, nil
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
		in := updateProgramInput()
		in.Status = ""

		_, err := app.NewUpdateDailyProgram(programRepo, zonesRepo, tracer()).
			Handle(t.Context(), app.UpdateDailyProgramCommand{UpdateProgramInput: in})

		require.NoError(t, err)
		require.Equal(t, program.EnabledStatus, current.Status())
	})
}

func TestUpdateOddAndEvenProgram_Handle(t *testing.T) {
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
			cmd:  app.UpdateOddProgramCommand{UpdateProgramInput: updateProgramInput()},
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
			cmd:  app.UpdateEvenProgramCommand{UpdateProgramInput: updateProgramInput()},
			newHandler: func(p app.ProgramRepository, z app.ZoneRepository) cqs.CommandHandler {
				return app.NewUpdateEvenProgram(p, z, tracer())
			},
		},
	}
	for _, tt := range tests {
		t.Run(`Given an update program command handler, when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			current := fixtures.ProgramBuilder{}.Build()
			programRepo := &ProgramRepositoryMock{
				FindByHourFunc: func(_ context.Context, _ *program.Hour) (*program.Program, error) {
					return &current, nil
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
			require.Len(t, programRepo.SaveCalls(), 1)
		})
	}
}

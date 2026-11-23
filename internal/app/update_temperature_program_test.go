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

func TestUpdateTemperatureProgram_Handle(t *testing.T) {
	errTest := errors.New("")
	defaultCmd := app.UpdateTemperatureProgramCommand{Temperature: 28.3, Programs: programInputs()}
	invalid := invalidProgramInputs()
	tests := []struct {
		name string
		cmd  cqs.Command
		expectedErr, findErr,
		saveErr error
	}{
		{
			name:        "with an invalid command, then it returns an invalid command error",
			cmd:         invalidCommand{},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:        "with an invalid hour, then it returns an update temperature program error",
			cmd:         app.UpdateTemperatureProgramCommand{Temperature: 28.3, Programs: invalid["hour"]},
			expectedErr: app.UpdateTemperatureProgramError{},
		},
		{
			name:        "with invalid seconds, then it returns an update temperature program error",
			cmd:         app.UpdateTemperatureProgramCommand{Temperature: 28.3, Programs: invalid["seconds"]},
			expectedErr: app.UpdateTemperatureProgramError{},
		},
		{
			name:        "with an invalid status, then it returns an update temperature program error",
			cmd:         app.UpdateTemperatureProgramCommand{Temperature: 28.3, Programs: invalid["status"]},
			expectedErr: app.UpdateTemperatureProgramError{},
		},
		{
			name:        "with a program without executions, then it returns an update temperature program error",
			cmd:         app.UpdateTemperatureProgramCommand{Temperature: 28.3, Programs: invalid["executions"]},
			expectedErr: app.UpdateTemperatureProgramError{},
		},
		{
			name:        "without programs, then it returns an update temperature program error",
			cmd:         app.UpdateTemperatureProgramCommand{Temperature: 28.3, Programs: invalid["programs"]},
			expectedErr: app.UpdateTemperatureProgramError{},
		},
		{
			name:        "and find program returns an error, then it returns same error",
			cmd:         defaultCmd,
			findErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and save program returns an error, then it returns same error",
			cmd:         defaultCmd,
			saveErr:     errTest,
			expectedErr: errTest,
		},
		{
			name: "and save program returns nil, then it saves the found temperature with the new programs",
			cmd:  defaultCmd,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an UpdateTemperatureProgram command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			temp := fixtures.TemperatureBuilder{}.Build()
			repo := &TemperatureProgramRepositoryMock{
				FindByTemperatureFunc: func(_ context.Context, _ float32) (*program.Temperature, error) {
					if tt.findErr != nil {
						return nil, tt.findErr
					}
					return &temp, nil
				},
				SaveFunc: func(_ context.Context, _ *program.Temperature) error {
					return tt.saveErr
				},
			}
			handler := app.NewUpdateTemperatureProgram(repo, tracer())

			_, err := handler.Handle(t.Context(), tt.cmd)
			if tt.expectedErr != nil {
				require.IsType(t, tt.expectedErr, err)
				if tt.saveErr == nil {
					require.Empty(t, repo.SaveCalls())
				}
				return
			}
			require.NoError(t, err)
			require.Len(t, repo.SaveCalls(), 1)
			saved := repo.SaveCalls()[0].ProgramMoqParam
			require.Same(t, &temp, saved)
			require.Len(t, saved.Programs(), 1)
			require.Equal(t, program.DisabledStatus, saved.Programs()[0].Status())
		})
	}
}

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

func programInputs() []app.ProgramInput {
	return []app.ProgramInput{
		{
			Hour:       "11:00",
			Status:     "disabled",
			Executions: []app.ProgramExecutionInput{{Seconds: 30, Zones: []string{"2"}}},
		},
	}
}

func invalidProgramInputs() map[string][]app.ProgramInput {
	execs := []app.ProgramExecutionInput{{Seconds: 30, Zones: []string{"2"}}}
	return map[string][]app.ProgramInput{
		"hour":       {{Hour: "invalid", Executions: execs}},
		"seconds":    {{Hour: "11:00", Executions: []app.ProgramExecutionInput{{Seconds: -1, Zones: []string{"2"}}}}},
		"status":     {{Hour: "11:00", Status: "paused", Executions: execs}},
		"executions": {{Hour: "11:00"}},
		"programs":   {},
	}
}

func TestUpdateWeeklyProgram_Handle(t *testing.T) {
	errTest := errors.New("")
	defaultCmd := app.UpdateWeeklyProgramCommand{WeekDay: "Friday", Programs: programInputs()}
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
			name:        "with an invalid week day, then it returns an update weekly program error",
			cmd:         app.UpdateWeeklyProgramCommand{WeekDay: "invalid", Programs: programInputs()},
			expectedErr: app.UpdateWeeklyProgramError{},
		},
		{
			name:        "with an invalid hour, then it returns an update weekly program error",
			cmd:         app.UpdateWeeklyProgramCommand{WeekDay: "Friday", Programs: invalid["hour"]},
			expectedErr: app.UpdateWeeklyProgramError{},
		},
		{
			name:        "with invalid seconds, then it returns an update weekly program error",
			cmd:         app.UpdateWeeklyProgramCommand{WeekDay: "Friday", Programs: invalid["seconds"]},
			expectedErr: app.UpdateWeeklyProgramError{},
		},
		{
			name:        "with an invalid status, then it returns an update weekly program error",
			cmd:         app.UpdateWeeklyProgramCommand{WeekDay: "Friday", Programs: invalid["status"]},
			expectedErr: app.UpdateWeeklyProgramError{},
		},
		{
			name:        "with a program without executions, then it returns an update weekly program error",
			cmd:         app.UpdateWeeklyProgramCommand{WeekDay: "Friday", Programs: invalid["executions"]},
			expectedErr: app.UpdateWeeklyProgramError{},
		},
		{
			name:        "without programs, then it returns an update weekly program error",
			cmd:         app.UpdateWeeklyProgramCommand{WeekDay: "Friday", Programs: invalid["programs"]},
			expectedErr: app.UpdateWeeklyProgramError{},
		},
		{
			name:        "and find weekly program returns an error, then it returns same error",
			cmd:         defaultCmd,
			findErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and save method returns an error, then it returns same error",
			cmd:         defaultCmd,
			saveErr:     errTest,
			expectedErr: errTest,
		},
		{
			name: "and save method returns nil, then it saves the found weekly with the new programs",
			cmd:  defaultCmd,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an UpdateWeeklyProgram command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			weekly := fixtures.WeeklyBuilder{}.Build()
			repo := &WeeklyProgramRepositoryMock{
				FindByDayFunc: func(_ context.Context, _ *program.WeekDay) (*program.Weekly, error) {
					if tt.findErr != nil {
						return nil, tt.findErr
					}
					return &weekly, nil
				},
				SaveFunc: func(_ context.Context, _ *program.Weekly) error {
					return tt.saveErr
				},
			}
			handler := app.NewUpdateWeeklyProgram(repo, tracer())

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
			require.Same(t, &weekly, saved)
			require.Len(t, saved.Programs(), 1)
			require.Equal(t, "11:00", saved.Programs()[0].Hour().String())
			require.Equal(t, program.DisabledStatus, saved.Programs()[0].Status())
		})
	}
}

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
	type args struct {
		ctx context.Context
		cmd cqs.Command
	}
	prg := fixtures.TemperatureBuilder{}.Build()

	defaultArgs := args{
		ctx: t.Context(),
		cmd: app.UpdateTemperatureProgramCommand{
			Temperature: prg.Temperature(),
			Programs:    prg.Programs(),
		},
	}

	tests := []struct {
		name string
		args args
		expectedErr, findErr,
		saveErr error
		program program.Temperature
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
			program:     prg,
			saveErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:    "and save program returns nil, then it returns nil",
			args:    defaultArgs,
			program: prg,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an UpdateTemperatureProgram command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &TemperatureProgramRepositoryMock{}
			repo.FindByTemperatureFunc = func(_ context.Context, _ float32) (*program.Temperature, error) {
				return &tt.program, tt.findErr
			}
			repo.SaveFunc = func(_ context.Context, _ *program.Temperature) error {
				return tt.saveErr
			}
			handler := app.NewUpdateTemperatureProgram(repo, tracer())
			_, err := handler.Handle(tt.args.ctx, tt.args.cmd)
			if err != nil {
				require.IsType(t, tt.expectedErr, err)
			}
		})
	}
}

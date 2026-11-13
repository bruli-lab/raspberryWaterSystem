package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestChangeCleanStatus_Handle(t *testing.T) {
	errortest := errors.New("")
	type args struct {
		cmd cqs.Command
	}
	st := fixtures.StatusBuilder{}.Build()
	tests := []struct {
		name string
		args args
		expectedErr, findErr,
		updateErr error
		status status.Status
	}{
		{
			name: "with an invalid command, then it returns an invalid command error",
			args: args{
				cmd: invalidCommand{},
			},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name: "and find status return an error, then it returns same error",
			args: args{
				cmd: app.ChangeCleanStatusCmd{Clean: true},
			},
			findErr:     errortest,
			expectedErr: errortest,
		},
		{
			name: "and update status return an error, then it returns same error",
			args: args{
				cmd: app.ChangeCleanStatusCmd{Clean: true},
			},
			status:      st,
			updateErr:   errortest,
			expectedErr: errortest,
		},
		{
			name: "and update status return nil, then it returns nil",
			args: args{
				cmd: app.ChangeCleanStatusCmd{Clean: false},
			},
			status: st,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a ChangeCleanStatus command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &StatusRepositoryMock{}
			repo.FindFunc = func(_ context.Context) (status.Status, error) {
				return tt.status, tt.findErr
			}
			repo.UpdateFunc = func(_ context.Context, _ status.Status) error {
				return tt.updateErr
			}

			handler := app.NewChangeCleanStatus(repo, tracer())
			_, err := handler.Handle(t.Context(), tt.args.cmd)
			if err != nil {
				require.IsType(t, tt.expectedErr, err)
			}
		})
	}
}

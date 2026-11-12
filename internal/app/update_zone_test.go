package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestUpdateZone_Handle(t *testing.T) {
	errTest := errors.New("")
	ctx := context.Background()
	type args struct {
		ctx context.Context
		cmd cqs.Command
	}
	zon := fixtures.ZoneBuilder{}.Build()
	defaultArgs := args{
		ctx: ctx,
		cmd: app.UpdateZoneCommand{
			ID:       zon.Id(),
			ZoneName: zon.Name(),
			Relays:   []int{1, 2, 3},
		},
	}
	tests := []struct {
		name               string
		args               args
		expectedErr        error
		findErr, updateErr error
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
			name:        "and find user returns a not found error, then it returns an update zone error",
			args:        defaultArgs,
			findErr:     errs.NotFoundError{},
			expectedErr: app.UpdateZoneError{},
		},
		{
			name:        "and find user returns an error, then it returns same error",
			args:        defaultArgs,
			findErr:     errTest,
			expectedErr: errTest,
		},
		{
			name: "and new user returns an error, then it returns an update zone error",
			args: args{
				ctx: ctx,
				cmd: app.UpdateZoneCommand{
					ID:       "dafa",
					ZoneName: "",
					Relays:   nil,
				},
			},
			expectedErr: app.UpdateZoneError{},
		},
		{
			name:        "and update returns an error, then it returns same error",
			args:        defaultArgs,
			updateErr:   errTest,
			expectedErr: errTest,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a UpdateZone command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &ZoneRepositoryMock{
				FindByIDFunc: func(_ context.Context, _ string) (*zone.Zone, error) {
					return nil, tt.findErr
				},
				UpdateFunc: func(_ context.Context, _ *zone.Zone) error {
					return tt.updateErr
				},
			}

			handler := app.NewUpdateZone(repo, tracer())
			_, err := handler.Handle(tt.args.ctx, tt.args.cmd)
			if err != nil {
				require.ErrorAs(t, err, &tt.expectedErr)
			}
		})
	}
}

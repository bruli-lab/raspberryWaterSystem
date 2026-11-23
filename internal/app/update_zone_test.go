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
			name: "with an invalid relay, then it returns an update zone error",
			args: args{
				ctx: ctx,
				cmd: app.UpdateZoneCommand{ID: zon.Id(), ZoneName: zon.Name(), Relays: []int{99}},
			},
			expectedErr: app.UpdateZoneError{},
		},
		{
			name: "and zone update returns an error, then it returns an update zone error",
			args: args{
				ctx: ctx,
				cmd: app.UpdateZoneCommand{
					ID:       zon.Id(),
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
		{
			name: "and update returns nil, then it updates the found zone",
			args: args{
				ctx: ctx,
				cmd: app.UpdateZoneCommand{ID: zon.Id(), ZoneName: "new name", Relays: []int{2}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a UpdateZone command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			current := fixtures.ZoneBuilder{}.Build()
			repo := &ZoneRepositoryMock{
				FindByIDFunc: func(_ context.Context, _ string) (*zone.Zone, error) {
					if tt.findErr != nil {
						return nil, tt.findErr
					}
					return current, nil
				},
				UpdateFunc: func(_ context.Context, _ *zone.Zone) error {
					return tt.updateErr
				},
			}

			handler := app.NewUpdateZone(repo, tracer())
			_, err := handler.Handle(tt.args.ctx, tt.args.cmd)
			if tt.expectedErr != nil {
				require.IsType(t, tt.expectedErr, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, repo.UpdateCalls(), 1)
			updated := repo.UpdateCalls()[0].Zo
			require.Same(t, current, updated)
			require.Equal(t, "new name", updated.Name())
			require.Len(t, updated.Relays(), 1)
		})
	}
}

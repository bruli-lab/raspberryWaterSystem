package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"

	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/stretchr/testify/require"
)

func TestRemoveZoneHandle(t *testing.T) {
	errTest := errors.New("")
	zo := fixtures.ZoneBuilder{}.Build()
	cmd := app.RemoveZoneCmd{}
	tests := []struct {
		name string
		expectedErr, findErr,
		removeErr error
		zone *zone.Zone
		cmd  cqs.Command
	}{
		{
			name:        "with an invalid command, then it returns an invalid command error",
			cmd:         invalidCommand{},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:        "and FindByID returns an error, then it returns same error",
			findErr:     errTest,
			expectedErr: errTest,
			cmd:         cmd,
		},
		{
			name:        "and remove method returns an error, then it returns same error",
			zone:        zo,
			removeErr:   errTest,
			expectedErr: errTest,
			cmd:         cmd,
		},
		{
			name: "then it returns any error",
			zone: zo,
			cmd:  cmd,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a RemoveZone command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			zr := &ZoneRepositoryMock{
				FindByIDFunc: func(_ context.Context, _ string) (*zone.Zone, error) {
					return tt.zone, tt.findErr
				},
				RemoveFunc: func(_ context.Context, _ *zone.Zone) error {
					return tt.removeErr
				},
			}
			handler := app.NewRemoveZone(zr, tracer())
			events, err := handler.Handle(context.Background(), tt.cmd)
			if err != nil {
				require.ErrorAs(t, err, &tt.expectedErr)
				return
			}
			require.Equal(t, tt.expectedErr, err)
			require.Nil(t, events)
		})
	}
}

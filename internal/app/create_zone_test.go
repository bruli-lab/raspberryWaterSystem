package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/stretchr/testify/require"

	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"

	"github.com/bruli/raspberryWaterSystem/internal/app"
)

func TestCreateZoneHandle(t *testing.T) {
	errTest := errors.New("")
	cmd := app.CreateZoneCmd{
		ID:       "id",
		ZoneName: "name",
		Relays:   []int{1},
	}
	tests := []struct {
		name string
		cmd  cqs.Command
		expectedErr, relayErr,
		zoneErr, saveErr, updateErr error
		zone *zone.Zone
	}{
		{
			name:        "with an invalid command, then it returns an invalid command error",
			cmd:         invalidCommand{},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:        "and find returns nil error, then it returns a create zone error",
			cmd:         cmd,
			expectedErr: app.CreateZoneError{},
		},
		{
			name:        "and find returns an error, then it returns same error",
			cmd:         cmd,
			zoneErr:     errTest,
			expectedErr: errTest,
		},
		{
			name: "with invalid relays, then it returns a create zone error",
			cmd: app.CreateZoneCmd{
				Relays: []int{99},
			},
			zoneErr:     errs.NotFoundError{},
			expectedErr: app.CreateZoneError{},
		},
		{
			name: "with invalid id, then it returns a create zone error",
			cmd: app.CreateZoneCmd{
				Relays: []int{1},
			},
			zoneErr:     errs.NotFoundError{},
			expectedErr: app.CreateZoneError{},
		},
		{
			name:        "and save method returns an error, then it returns same error",
			cmd:         cmd,
			zoneErr:     errs.NotFoundError{},
			saveErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:    "then it returns return nil",
			cmd:     cmd,
			zoneErr: errs.NotFoundError{},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a CreateZone command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			zr := &ZoneRepositoryMock{
				FindByIDFunc: func(_ context.Context, _ string) (*zone.Zone, error) {
					return tt.zone, tt.zoneErr
				},
				SaveFunc: func(_ context.Context, _ *zone.Zone) error {
					return tt.saveErr
				},
			}
			handler := app.NewCreateZone(zr, tracer())
			_, errHand := handler.Handle(context.Background(), tt.cmd)
			if errHand != nil {
				require.IsType(t, tt.expectedErr, errHand)
			}
		})
	}
}

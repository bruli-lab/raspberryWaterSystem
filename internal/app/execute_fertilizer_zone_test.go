package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestExecuteFertilizerZoneHandle(t *testing.T) {
	errTest := errors.New("")
	zo := fixtures.ZoneBuilder{}.Build()
	cmd := app.ExecuteFertilizerZoneCmd{
		Seconds: 367,
		ZoneID:  "name",
	}
	tests := []struct {
		name    string
		command cqs.Command
		expectedErr, findErr,
		findStErr error
		zone   *zone.Zone
		status status.Status
	}{
		{
			name:        "with invalid command, then it returns an invalid command error",
			command:     invalidCommand{},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:        "and find status returns an error, then it returns same error",
			command:     cmd,
			findStErr:   errTest,
			expectedErr: errTest,
		},
		{
			name:        "and find status returns an status with clean, then it returns execute fertilizer zone error",
			command:     cmd,
			status:      fixtures.StatusBuilder{Clean: true}.Build(),
			expectedErr: app.ExecuteFertilizerZoneError{},
		},
		{
			name:        "and find zone returns an error, then it returns same error",
			command:     cmd,
			findErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and execute returns an error, then it returns an execute fertilizer zone error",
			zone:        zo,
			expectedErr: app.ExecuteFertilizerZoneError{},
			command:     cmd,
		},
		{
			name: "and execute nil, then it returns a valid event",
			zone: zo,
			command: app.ExecuteFertilizerZoneCmd{
				Seconds: 36,
				ZoneID:  "name",
			},
		},
	}
	for _, tt := range tests {
		t.Run(`Given an executeFertilizerZone command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			zr := &ZoneRepositoryMock{
				FindByIDFunc: func(_ context.Context, _ string) (*zone.Zone, error) {
					return tt.zone, tt.findErr
				},
			}
			str := &StatusRepositoryMock{
				FindFunc: func(_ context.Context) (status.Status, error) {
					return tt.status, tt.findStErr
				},
			}
			handler := app.NewExecuteFertilizerZone(zr, str, tracer())
			events, err := handler.Handle(context.Background(), tt.command)
			if err != nil {
				require.ErrorAs(t, err, &tt.expectedErr)
				return
			}
			require.Equal(t, tt.expectedErr, err)
			require.Len(t, events, 1)
			ev := events[0]
			_, ok := ev.(zone.FertilizerZoneExecuted)
			require.True(t, ok)
		})
	}
}

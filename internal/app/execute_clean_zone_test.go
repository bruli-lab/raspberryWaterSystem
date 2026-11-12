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

func TestExecuteCleanZoneHandle(t *testing.T) {
	errTest := errors.New("")
	zo := fixtures.ZoneBuilder{}.Build()
	cmd := app.ExecuteCleanZoneCmd{
		ZoneID: "name",
	}
	tests := []struct {
		name                                         string
		command                                      cqs.Command
		expectedErr, findErr, findStErr, updateStErr error
		zone                                         *zone.Zone
		st                                           status.Status
		expectedUpdates                              int
	}{
		{
			name:        "with invalid command, then it returns an invalid command error",
			command:     invalidCommand{},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:        "and find zone returns an error, then it returns same error",
			command:     cmd,
			findErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and find status returns an error, then it returns same error",
			command:     cmd,
			zone:        zo,
			findStErr:   errTest,
			expectedErr: errTest,
		},
		{
			name:        "and a clean cycle is already in progress, then it returns a clean in progress error",
			command:     cmd,
			zone:        zo,
			st:          fixtures.StatusBuilder{Active: true, Clean: true}.Build(),
			expectedErr: app.CleanInProgressError{},
		},
		{
			name:            "and update status returns an error, then it returns same error",
			command:         cmd,
			zone:            zo,
			st:              fixtures.StatusBuilder{Active: true}.Build(),
			updateStErr:     errTest,
			expectedErr:     errTest,
			expectedUpdates: 1,
		},
		{
			name:            "and execute nil, then it marks the status as cleaning and returns a valid event",
			command:         cmd,
			zone:            zo,
			st:              fixtures.StatusBuilder{Active: true}.Build(),
			expectedUpdates: 1,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an executeCleanZone command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			zr := &ZoneRepositoryMock{
				FindByIDFunc: func(_ context.Context, _ string) (*zone.Zone, error) {
					return tt.zone, tt.findErr
				},
			}
			sr := &StatusRepositoryMock{
				FindFunc: func(_ context.Context) (status.Status, error) {
					return tt.st, tt.findStErr
				},
				UpdateFunc: func(_ context.Context, st status.Status) error {
					require.True(t, st.IsClean())
					return tt.updateStErr
				},
			}
			handler := app.NewExecuteCleanZone(zr, sr, tracer())
			events, err := handler.Handle(context.Background(), tt.command)
			require.Len(t, sr.UpdateCalls(), tt.expectedUpdates)
			if tt.expectedErr != nil {
				require.ErrorAs(t, err, &tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, events, 1)
			_, ok := events[0].(zone.CleanZoneExecuted)
			require.True(t, ok)
		})
	}
}

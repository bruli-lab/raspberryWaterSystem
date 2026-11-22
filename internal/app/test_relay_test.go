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

func TestTestRelayHandle(t *testing.T) {
	errTest := errors.New("")
	idle := fixtures.StatusBuilder{Active: true}.Build()
	cleaning := fixtures.StatusBuilder{Active: true, Clean: true}.Build()
	tests := []struct {
		name                          string
		execErr, findErr, expectedErr error
		st                            status.Status
		cmd                           cqs.Command
		expectedPins                  []string
	}{
		{
			name:        "with an invalid command, then it returns an invalid command error",
			cmd:         invalidCommand{},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:        "with an unknown relay, then it returns an unknown relay error without executing pins",
			cmd:         app.TestRelayCmd{RelayID: 99},
			st:          idle,
			expectedErr: zone.ErrUnknownRelay,
		},
		{
			name:        "and find status returns an error, then it returns same error without executing pins",
			cmd:         app.TestRelayCmd{RelayID: zone.OneRelayID},
			findErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and a clean cycle is in progress, then it returns a clean in progress error without executing pins",
			cmd:         app.TestRelayCmd{RelayID: zone.OneRelayID},
			st:          cleaning,
			expectedErr: app.CleanInProgressError{},
		},
		{
			name:         "and executor returns an error, then it returns same error",
			cmd:          app.TestRelayCmd{RelayID: zone.OneRelayID},
			st:           idle,
			execErr:      errTest,
			expectedErr:  errTest,
			expectedPins: []string{"18"},
		},
		{
			name:         "with a zone relay, then it activates its pin during the test seconds and returns no events",
			cmd:          app.TestRelayCmd{RelayID: zone.OneRelayID},
			st:           idle,
			expectedPins: []string{"18"},
		},
		{
			name:         "with a fixed relay, then it activates its pin during the test seconds and returns no events",
			cmd:          app.TestRelayCmd{RelayID: zone.FertilizerPumpID},
			st:           idle,
			expectedPins: []string{"22"},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a TestRelay command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			pe := &PinExecutorMock{
				ExecuteFunc: func(_ context.Context, _ uint, _ []string) error {
					return tt.execErr
				},
			}
			sr := &StatusRepositoryMock{
				FindFunc: func(_ context.Context) (status.Status, error) {
					return tt.st, tt.findErr
				},
			}
			handler := app.NewTestRelay(pe, sr, tracer())
			events, err := handler.Handle(context.Background(), tt.cmd)
			if tt.expectedPins == nil {
				require.Empty(t, pe.ExecuteCalls())
			} else {
				require.Len(t, pe.ExecuteCalls(), 1)
				require.Equal(t, uint(app.RelayTestSeconds), pe.ExecuteCalls()[0].Seconds)
				require.Equal(t, tt.expectedPins, pe.ExecuteCalls()[0].Pins)
			}
			if tt.expectedErr != nil {
				require.IsType(t, tt.expectedErr, err)
				return
			}
			require.NoError(t, err)
			require.Nil(t, events)
		})
	}
}

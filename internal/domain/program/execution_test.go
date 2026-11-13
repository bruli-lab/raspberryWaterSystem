package program_test

import (
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/stretchr/testify/require"
)

func TestNewExecution(t *testing.T) {
	tests := []struct {
		name        string
		seconds     program.Seconds
		zones       []string
		expectedErr error
	}{
		{
			name:        "with empty zones, then returns an empty execution zones error",
			seconds:     program.HydrateSeconds(20),
			expectedErr: program.ErrEmptyExecutionZones,
		},
		{
			name:        "with seconds over the limit, then returns an invalid seconds error",
			seconds:     program.HydrateSeconds(121),
			zones:       []string{"1"},
			expectedErr: program.InvalidSecondsError{},
		},
		{
			name:        "with zero seconds, then returns an invalid seconds error",
			seconds:     program.HydrateSeconds(0),
			zones:       []string{"1"},
			expectedErr: program.InvalidSecondsError{},
		},
		{
			name:    "with all values, then returns a valid struct",
			seconds: program.HydrateSeconds(20),
			zones:   []string{"1", "2"},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a Execution struct,
		when NewExecution function is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			exec, err := program.NewExecution(tt.seconds, tt.zones)
			if tt.expectedErr != nil {
				requireDomainError(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.seconds, exec.Seconds())
			require.Equal(t, tt.zones, exec.Zones())
		})
	}
}

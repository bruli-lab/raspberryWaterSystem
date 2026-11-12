package program_test

import (
	"errors"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/stretchr/testify/require"
)

func TestParseSeconds(t *testing.T) {
	tests := []struct {
		name             string
		seconds          int
		expectedErr      bool
		expectedDuration time.Duration
	}{
		{name: "with negative seconds, then it returns an invalid seconds error", seconds: -1, expectedErr: true},
		{name: "with zero seconds, then it returns an invalid seconds error", seconds: 0, expectedErr: true},
		{name: "with seconds over the limit, then it returns an invalid seconds error", seconds: 121, expectedErr: true},
		{name: "with the minimum seconds, then it returns its duration", seconds: 1, expectedDuration: time.Second},
		{name: "with the maximum seconds, then it returns its duration", seconds: 120, expectedDuration: 2 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(`Given a seconds value,
		when ParseSeconds is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			sec, err := program.ParseSeconds(tt.seconds)
			if tt.expectedErr {
				require.ErrorAs(t, err, &program.InvalidSecondsError{})
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expectedDuration, sec.Duration())
			require.Equal(t, tt.seconds, sec.Int())
		})
	}
}

func requireDomainError(t *testing.T, err, expected error) {
	t.Helper()
	if errors.As(expected, &program.InvalidSecondsError{}) {
		require.ErrorAs(t, err, &program.InvalidSecondsError{})
		return
	}
	require.ErrorIs(t, err, expected)
}

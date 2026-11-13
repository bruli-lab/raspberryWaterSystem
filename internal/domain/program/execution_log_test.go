package program_test

import (
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/stretchr/testify/require"
)

func TestNewExecutionLog(t *testing.T) {
	seconds, _ := program.ParseSeconds(20)
	zone := "zone name"
	tests := []struct {
		name        string
		seconds     program.Seconds
		zoneName    string
		executedAt  time.Time
		expectedErr error
	}{
		{
			name:        "with invalid seconds, then it returns an invalid seconds error",
			seconds:     program.Seconds(-1 * time.Second),
			expectedErr: program.InvalidSecondsError{},
		},
		{
			name:        "with invalid zone name, then it returns a empty zone name error",
			seconds:     seconds,
			zoneName:    "",
			expectedErr: program.ErrEmptyZoneName,
		},
		{
			name:        "with invalid executed at, then it returns a invalid executed at error",
			seconds:     seconds,
			zoneName:    zone,
			executedAt:  time.Time{},
			expectedErr: program.ErrInvalidExecutedAt,
		},
		{
			name:       "with all values, then it returns a valid struct",
			seconds:    seconds,
			zoneName:   zone,
			executedAt: time.Now(),
		},
	}
	for _, tt := range tests {
		t.Run(`Given a executionLog struct,
		when NewExecutionLog function is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			execLog, err := program.NewExecutionLog(tt.seconds, tt.zoneName, tt.executedAt)
			if tt.expectedErr != nil {
				requireDomainError(t, err, tt.expectedErr)
				return
			}
			require.Equal(t, tt.expectedErr, err)
			require.Equal(t, tt.seconds, execLog.Seconds())
			require.Equal(t, tt.zoneName, execLog.ZoneName())
			require.Equal(t, tt.executedAt, execLog.ExecutedAt())
		})
	}
}

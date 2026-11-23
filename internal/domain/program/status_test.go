package program_test

import (
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/stretchr/testify/require"
)

func TestParseStatus(t *testing.T) {
	tests := []struct {
		name, value     string
		expectedErr     error
		expectedStatus  program.Status
		expectedEnabled bool
	}{
		{
			name:        "with an invalid value, then it returns an invalid program status error",
			value:       "paused",
			expectedErr: program.ErrInvalidStatus,
		},
		{
			name:        "with an empty value, then it returns an invalid program status error",
			expectedErr: program.ErrInvalidStatus,
		},
		{
			name:            "with the enabled value, then it returns an enabled status",
			value:           "enabled",
			expectedStatus:  program.EnabledStatus,
			expectedEnabled: true,
		},
		{
			name:           "with the disabled value, then it returns a disabled status",
			value:          "disabled",
			expectedStatus: program.DisabledStatus,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a Status type,
		when ParseStatus function is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			st, err := program.ParseStatus(tt.value)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expectedStatus, st)
			require.Equal(t, tt.value, st.String())
			require.Equal(t, tt.expectedEnabled, st.IsEnabled())
		})
	}
}

func TestProgramSetStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         program.Status
		expectedStatus program.Status
	}{
		{
			name:           "with a disabled status, then the program is disabled",
			status:         program.DisabledStatus,
			expectedStatus: program.DisabledStatus,
		},
		{
			name:           "with an enabled status, then the program is enabled",
			status:         program.EnabledStatus,
			expectedStatus: program.EnabledStatus,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a new Program, when SetStatus method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			hour, err := program.ParseHour("10:00")
			require.NoError(t, err)
			sec, err := program.ParseSeconds(10)
			require.NoError(t, err)
			exec, err := program.NewExecution(sec, []string{"1"})
			require.NoError(t, err)
			prg, err := program.New(hour, false, []program.Execution{exec})
			require.NoError(t, err)

			prg.SetStatus(tt.status)

			require.Equal(t, tt.expectedStatus, prg.Status())
		})
	}
}

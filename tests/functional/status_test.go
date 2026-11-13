//go:build functional

package functional_test

import (
	http2 "net/http"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/bruli/raspberryWaterSystem/tests/harness"
	"github.com/stretchr/testify/require"
)

func TestStatus(t *testing.T) {
	tests := []struct {
		name           string
		actions        []string
		expectedActive bool
	}{
		{
			name:           "the status is requested, then it is active and has a start time",
			expectedActive: true,
		},
		{
			name:    "the server is deactivated, then the status is not active",
			actions: []string{"deactivate"},
		},
		{
			name:           "the server is deactivated and activated again, then the status is active",
			actions:        []string{"deactivate", "activate"},
			expectedActive: true,
		},
	}
	for _, tt := range tests {
		t.Run(`Given the app running,
		when `+tt.name, func(t *testing.T) {
			t.Parallel()
			st := harness.Start(t, appBinary)
			for _, action := range tt.actions {
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodPatch, "/status/"+action, nil))
			}

			var status http.StatusResponse
			st.Get(t, "/status", &status)

			require.NotEmpty(t, status.SystemStartedAt)
			require.Equal(t, tt.expectedActive, status.Active)
		})
	}
}

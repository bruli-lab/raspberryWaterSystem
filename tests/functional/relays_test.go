//go:build functional

package functional_test

import (
	http2 "net/http"
	"testing"

	"github.com/bruli/raspberryWaterSystem/tests/harness"
	"github.com/stretchr/testify/require"
)

func TestRelays(t *testing.T) {
	st := harness.StartWithDependenciesDown(t, appBinary)
	tests := []struct {
		name, id     string
		expectedCode int
	}{
		{name: "of a zone, then it returns ok", id: "1", expectedCode: http2.StatusOK},
		{name: "of the fertilizer pump, then it returns ok", id: "5", expectedCode: http2.StatusOK},
		{name: "that does not exist, then it returns bad request", id: "99", expectedCode: http2.StatusBadRequest},
		{name: "with a non numeric id, then it returns bad request", id: "pump", expectedCode: http2.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(`Given the app running,
		when a relay is tested `+tt.name+` and it leaves no execution log, event or notification`, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expectedCode, st.Send(t, http2.MethodPost, "/relays/"+tt.id+"/test", nil))
			require.Empty(t, st.ExecutionLogs(t))
			require.Zero(t, st.EventsInOutbox(t))
			require.Zero(t, st.NotificationsInOutbox(t))
		})
	}
}

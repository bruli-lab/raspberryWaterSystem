//go:build resilience

package resilience_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	infrahttp "github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/bruli/raspberryWaterSystem/tests/harness"
	"github.com/stretchr/testify/require"
)

const outboxPeriod = 2 * time.Minute

var appBinary string

func TestMain(m *testing.M) {
	binary, cleanup, err := harness.BuildApp(context.Background())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	appBinary = binary
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func TestIrrigationWithoutNatsAndNtfy(t *testing.T) {
	tests := []struct {
		name, zoneID, zoneName string
		seconds                int
		expectedNotification   string
	}{
		{
			name:                 "then the zone is watered, the execution log is saved and the outbox is delivered when nats and ntfy come back",
			zoneID:               "bbf",
			zoneName:             "bonsai big",
			seconds:              1,
			expectedNotification: "bonsai big zone executed during 1s",
		},
	}
	for _, tt := range tests {
		t.Run(`Given the app started while nats and ntfy are down,
		when a zone is executed `+tt.name, func(t *testing.T) {
			t.Parallel()
			st := harness.StartWithDependenciesDown(t, appBinary)
			st.CreateZone(t, infrahttp.Zone{Id: tt.zoneID, Name: tt.zoneName, Relays: []infrahttp.ZoneRelays{1, 2}})

			require.Equal(t, http.StatusOK, st.Send(t, http.MethodPost, "/zones/"+tt.zoneID+"/execute", infrahttp.ExecuteZoneRequest{Seconds: tt.seconds}))

			require.Eventually(t, func() bool {
				logs := st.ExecutionLogs(t)
				return len(logs) == 1 && logs[0].ZoneName == tt.zoneName && logs[0].Seconds == tt.seconds
			}, 15*time.Second, 200*time.Millisecond, "execution log not saved while offline")
			require.Eventually(t, func() bool {
				return st.EventsInOutbox(t) == 1 && st.NotificationsInOutbox(t) == 1
			}, 5*time.Second, 100*time.Millisecond, "outbox did not keep the event and the notification while offline")

			st.StartDependencies(t)

			require.Eventually(t, func() bool {
				return st.EventsInOutbox(t) == 0 && st.NotificationsInOutbox(t) == 0
			}, outboxPeriod, time.Second, "outbox not emptied after nats and ntfy came back")
			published := st.LastPublishedExecutionLog(t)
			require.Equal(t, tt.zoneName, published.ZoneName)
			require.Equal(t, tt.seconds, published.Seconds)
			require.Contains(t, st.NtfyMessages(t), tt.expectedNotification)
		})
	}
}

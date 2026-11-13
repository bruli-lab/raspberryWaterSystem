//go:build functional

package functional_test

import (
	"fmt"
	http2 "net/http"
	"slices"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/bruli/raspberryWaterSystem/tests/harness"
	"github.com/stretchr/testify/require"
)

func zoneIDs(t *testing.T, st *harness.Stack) []string {
	t.Helper()
	var zones []http.Zone
	st.Get(t, "/zones", &zones)
	ids := make([]string, len(zones))
	for i, zon := range zones {
		ids[i] = zon.Id
	}
	return ids
}

func TestZones(t *testing.T) {
	st := harness.Start(t, appBinary)
	tests := []struct {
		name string
		run  func(t *testing.T, zone http.Zone)
	}{
		{
			name: "is created, then it is listed with its relays",
			run: func(t *testing.T, zone http.Zone) {
				t.Helper()
				var zones []http.Zone
				st.Get(t, "/zones", &zones)
				require.Contains(t, zones, zone)
			},
		},
		{
			name: "is executed, then it returns ok and the execution log is saved",
			run: func(t *testing.T, zone http.Zone) {
				t.Helper()
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodPost, "/zones/"+zone.Id+"/execute", http.ExecuteZoneRequest{Seconds: 1}))
				require.Eventually(t, func() bool {
					return slices.ContainsFunc(st.ExecutionLogs(t), func(log http.ExecutionLogsResponse) bool {
						return log.ZoneName == zone.Name && log.Seconds == 1
					})
				}, 15*time.Second, 200*time.Millisecond)
			},
		},
		{
			name: "is updated, then it is listed with the new name and relays",
			run: func(t *testing.T, zone http.Zone) {
				t.Helper()
				update := http.UpdateZoneRequest{Name: "updated " + zone.Name, Relays: []http.UpdateZoneRequestRelays{4}}
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodPut, "/zones/"+zone.Id, update))
				var zones []http.Zone
				st.Get(t, "/zones", &zones)
				require.Contains(t, zones, http.Zone{Id: zone.Id, Name: update.Name, Relays: []http.ZoneRelays{4}})
			},
		},
		{
			name: "is removed, then it is no longer listed",
			run: func(t *testing.T, zone http.Zone) {
				t.Helper()
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodDelete, "/zones/"+zone.Id, nil))
				require.NotContains(t, zoneIDs(t, st), zone.Id)
			},
		},
		{
			name: "does not exist and it is executed, then it returns not found",
			run: func(t *testing.T, _ http.Zone) {
				t.Helper()
				require.Equal(t, http2.StatusNotFound, st.Send(t, http2.MethodPost, "/zones/unknown/execute", http.ExecuteZoneRequest{Seconds: 1}))
			},
		},
	}
	for _, tt := range tests {
		t.Run(`Given the app running,
		when a zone `+tt.name, func(t *testing.T) {
			t.Parallel()
			zone := newZone()
			st.CreateZone(t, zone)
			tt.run(t, zone)
		})
	}
}

func TestSpecialZoneExecutions(t *testing.T) {
	tests := []struct {
		name, path string
		body       any
	}{
		{
			name: "with fertilizer, then it returns ok",
			path: "/zones/fertilizer/%s/execute",
			body: http.ExecuteZoneRequest{Seconds: 5},
		},
		{
			name: "with a clean cycle, then it returns ok",
			path: "/zones/clean/%s/execute",
		},
	}
	for _, tt := range tests {
		t.Run(`Given the app running with a zone,
		when the zone is executed `+tt.name, func(t *testing.T) {
			t.Parallel()
			st := harness.Start(t, appBinary)
			zone := newZone()
			st.CreateZone(t, zone)

			require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodPost, fmt.Sprintf(tt.path, zone.Id), tt.body))
		})
	}
}

//go:build functional

package functional_test

import (
	http2 "net/http"
	"slices"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/bruli/raspberryWaterSystem/tests/harness"
	"github.com/stretchr/testify/require"
)

func programs(t *testing.T, st *harness.Stack) http.ProgramsResponse {
	t.Helper()
	var resp http.ProgramsResponse
	st.Get(t, "/programs", &resp)
	return resp
}

func hasHour(items []http.ProgramItemResponse, hour string) bool {
	return slices.ContainsFunc(items, func(item http.ProgramItemResponse) bool {
		return item.Hour == hour
	})
}

func TestPrograms(t *testing.T) {
	st := harness.Start(t, appBinary)
	tests := []struct {
		name string
		run  func(t *testing.T, zoneID string)
	}{
		{
			name: "a daily program is created, then it is listed",
			run: func(t *testing.T, zoneID string) {
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodPost, "/programs/daily", program("12:45", zoneID)))
				require.True(t, hasHour(programs(t, st).Daily, "12:45"))
			},
		},
		{
			name: "a daily program is removed, then it is no longer listed",
			run: func(t *testing.T, zoneID string) {
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodPost, "/programs/daily", program("13:45", zoneID)))
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodDelete, "/programs/daily/13:45", nil))
				require.False(t, hasHour(programs(t, st).Daily, "13:45"))
			},
		},
		{
			name: "an odd program is created and removed, then it is no longer listed",
			run: func(t *testing.T, zoneID string) {
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodPost, "/programs/odd", program("14:45", zoneID)))
				require.True(t, hasHour(programs(t, st).Odd, "14:45"))
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodDelete, "/programs/odd/14:45", nil))
				require.False(t, hasHour(programs(t, st).Odd, "14:45"))
			},
		},
		{
			name: "an even program is created and removed, then it is no longer listed",
			run: func(t *testing.T, zoneID string) {
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodPost, "/programs/even", program("15:45", zoneID)))
				require.True(t, hasHour(programs(t, st).Even, "15:45"))
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodDelete, "/programs/even/15:45", nil))
				require.False(t, hasHour(programs(t, st).Even, "15:45"))
			},
		},
		{
			name: "a daily program that does not exist is removed, then it returns not found",
			run: func(t *testing.T, _ string) {
				require.Equal(t, http2.StatusNotFound, st.Send(t, http2.MethodDelete, "/programs/daily/23:59", nil))
			},
		},
		{
			name: "a weekly program is created and removed, then it is no longer listed",
			run: func(t *testing.T, zoneID string) {
				weekly := http.WeeklyProgramRequest{Programs: []http.Program{program("10:00", zoneID)}, WeekDay: "Monday"}
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodPost, "/programs/weekly", weekly))
				require.True(t, slices.ContainsFunc(programs(t, st).Weekly, func(item http.WeeklyItemResponse) bool {
					return item.WeekDay == "Monday" && hasHour(item.Programs, "10:00")
				}))
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodDelete, "/programs/weekly/Monday", nil))
				require.False(t, slices.ContainsFunc(programs(t, st).Weekly, func(item http.WeeklyItemResponse) bool {
					return item.WeekDay == "Monday"
				}))
			},
		},
		{
			name: "a temperature program is created, updated and removed, then each change is listed",
			run: func(t *testing.T, zoneID string) {
				temperature := func(hour string) bool {
					return slices.ContainsFunc(programs(t, st).Temperature, func(item http.TemperatureItemResponse) bool {
						return item.Temperature == 20 && (hour == "" || hasHour(item.Programs, hour))
					})
				}
				create := http.TemperatureProgramRequest{Programs: []http.Program{program("10:00", zoneID)}, Temperature: 20}
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodPost, "/programs/temperature", create))
				require.True(t, temperature("10:00"))
				update := http.UpdateTemperatureProgramRequest{Programs: []http.Program{program("17:00", zoneID)}}
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodPut, "/programs/temperature/20", update))
				require.True(t, temperature("17:00"))
				require.Equal(t, http2.StatusOK, st.Send(t, http2.MethodDelete, "/programs/temperature/20", nil))
				require.False(t, temperature(""))
			},
		},
	}
	for _, tt := range tests {
		t.Run(`Given the app running with a zone,
		when `+tt.name, func(t *testing.T) {
			t.Parallel()
			zone := newZone()
			st.CreateZone(t, zone)
			tt.run(t, zone.Id)
		})
	}
}

func program(hour, zoneID string) http.Program {
	return http.Program{
		Executions: []http.ExecutionRequest{{Seconds: 10, Zones: []string{zoneID}}},
		Hour:       hour,
	}
}

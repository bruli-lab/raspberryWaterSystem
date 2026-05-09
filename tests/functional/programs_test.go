//go:build functional

package functional

import (
	http2 "net/http"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/http"

	"github.com/stretchr/testify/require"
)

func TestPrograms(t *testing.T) {
	zo := fixtures.ZoneBuilder{}.Build()
	relays := make([]int, len(zo.Relays()))
	for i, r := range zo.Relays() {
		relays[i] = r.Id().Int()
	}
	req := http.CreateZoneRequestJson{
		Id:     zo.Id(),
		Name:   zo.Name(),
		Relays: relays,
	}
	resp, err := buildRequestAndSend(ctx, req, authorizationHeader(), http2.MethodPost, "/zones", cl)
	require.NoError(t, err)
	require.Equal(t, http2.StatusOK, resp.StatusCode)
	t.Run(`Given a create daily program endpoint,
	when a request is sent`, func(t *testing.T) {
		t.Run(`without authorization,
		then it returns unauthorized`, func(t *testing.T) {
			resp, err = buildRequestAndSend(ctx, nil, nil, http2.MethodPost, "/programs/daily", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`with authorization,
		then it returns ok`, func(t *testing.T) {
			createPrReq := http.CreateProgramRequestJson{
				Executions: []http.ExecutionRequest{
					{
						Seconds: 10,
						Zones:   []string{zo.Id()},
					},
				},
				Hour: "12:45",
			}
			resp, err = buildRequestAndSend(ctx, createPrReq, authorizationHeader(), http2.MethodPost, "/programs/daily", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusOK, resp.StatusCode)
		})
	})
	t.Run(`Given a delete daily program endpoint
	when a request is sent`, func(t *testing.T) {
		t.Run(`without authorization,
			then it returns unauthorized`, func(t *testing.T) {
			resp, err = buildRequestAndSend(ctx, nil, nil, http2.MethodDelete, "/programs/daily/12:45", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`with authorization,
			then it returns ok`, func(t *testing.T) {
			resp, err = buildRequestAndSend(ctx, nil, authorizationHeader(), http2.MethodDelete, "/programs/daily/12:45", cl)
			require.NoError(t, err)
		},
		)
	})
	t.Run(`Given a create weekly program endpoint,
	when a request is sent`, func(t *testing.T) {
		createWeekly := http.CreateWeeklyProgramRequestJson{
			Programs: []http.ProgramWeeklyRequest{
				{
					Executions: []http.ExecutionWeeklyRequest{
						{
							Seconds: 10,
							Zones:   []string{zo.Id()},
						},
					},
					Hour: "10:00",
				},
			},
			WeekDay: "Monday",
		}
		t.Run(`without authorization,
		then it returns an unauthorized`, func(t *testing.T) {
			resp, err = buildRequestAndSend(ctx, createWeekly, nil, http2.MethodPost, "/programs/weekly", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`with authorization,
		then it returns ok`, func(t *testing.T) {
			resp, err = buildRequestAndSend(ctx, createWeekly, authorizationHeader(), http2.MethodPost, "/programs/weekly", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusOK, resp.StatusCode)
		})
	})
	t.Run(`Given a create temperature program endpoint,
	when a request is sent`, func(t *testing.T) {
		createTemp := http.CreateTemperatureProgramRequestJson{
			Programs: []http.ProgramTemperatureRequest{
				{
					Executions: []http.ExecutionTemperatureRequest{
						{
							Seconds: 10,
							Zones:   []string{zo.Id()},
						},
					},
					Hour: "10:00",
				},
			},
			Temperature: 20,
		}
		t.Run(`without authorization,
		then it returns an unauthorized`, func(t *testing.T) {
			resp, err = buildRequestAndSend(ctx, createTemp, nil, http2.MethodPost, "/programs/temperature", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`with authorization,
		then it returns ok`, func(t *testing.T) {
			resp, err = buildRequestAndSend(ctx, createTemp, authorizationHeader(), http2.MethodPost, "/programs/temperature", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusOK, resp.StatusCode)
		})
	})
	t.Run(`Given a find all programs endpoint,
	when a request is sent`, func(t *testing.T) {
		t.Run(`without authorization,
		then it returns an unauthorized`, func(t *testing.T) {
			resp, err = buildRequestAndSend(ctx, nil, nil, http2.MethodGet, "/programs", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`with authorization,
		then it returns a valid response`, func(t *testing.T) {
			resp, err = buildRequestAndSend(ctx, nil, authorizationHeader(), http2.MethodGet, "/programs", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusOK, resp.StatusCode)
			var schema http.ProgramsResponseJson
			readResponse(t, resp, &schema)
		})
	})
	t.Run(`Given an update temperature program endpoint,
	when a request is sent`, func(t *testing.T) {
		t.Run(`without authorization,
		then it returns an authorized`, func(t *testing.T) {
			resp, err = buildRequestAndSend(ctx, nil, nil, http2.MethodPut, "/programs/temperature/20", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusUnauthorized, resp.StatusCode)
		})

		t.Run(`with authorization,
		then it returns an ok`, func(t *testing.T) {
			updateTemp := http.UpdateTemperatureProgramRequestJson{
				{
					Executions: []http.UpdateExecutionTemperatureRequest{
						{
							Seconds: 10,
							Zones:   []string{zo.Id()},
						},
					},
					Hour: "17:00",
				},
			}
			resp, err = buildRequestAndSend(ctx, updateTemp, authorizationHeader(), http2.MethodPut, "/programs/temperature/20", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusOK, resp.StatusCode)
		})
	})
	t.Run(`Given a remove weekly program endpoint,
	when a request is sent`, func(t *testing.T) {
		t.Run(`without authorization,
		then it returns an unauthorized`, func(t *testing.T) {
			resp, err = buildRequestAndSend(ctx, nil, nil, http2.MethodDelete, "/programs/weekly/Monday", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`with authorization,
		then it returns ok`, func(t *testing.T) {
			resp, err = buildRequestAndSend(ctx, nil, authorizationHeader(), http2.MethodDelete, "/programs/weekly/Monday", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusOK, resp.StatusCode)
		})
	})
	t.Run(`Given a remove temperature program endpoint,
	when a request is sent`, func(t *testing.T) {
		t.Run(`without authorization,
		then it returns an unauthorized`, func(t *testing.T) {
			resp, err = buildRequestAndSend(ctx, nil, nil, http2.MethodDelete, "/programs/temperature/20", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`with authorization,
		then it returns ok`, func(t *testing.T) {
			resp, err = buildRequestAndSend(ctx, nil, authorizationHeader(), http2.MethodDelete, "/programs/temperature/20", cl)
			require.NoError(t, err)
			require.Equal(t, http2.StatusOK, resp.StatusCode)
		})
	})
}

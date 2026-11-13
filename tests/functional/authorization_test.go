//go:build functional

package functional_test

import (
	"net/http"
	"testing"

	"github.com/bruli/raspberryWaterSystem/tests/harness"
	"github.com/stretchr/testify/require"
)

func TestEndpointsWithoutAuthorization(t *testing.T) {
	st := harness.Start(t, appBinary)
	tests := []struct {
		name, method, path string
		expectedCode       int
	}{
		{name: "to the homepage, then it returns ok", method: http.MethodGet, path: "/", expectedCode: http.StatusOK},
		{name: "to create a zone, then it returns unauthorized", method: http.MethodPost, path: "/zones", expectedCode: http.StatusUnauthorized},
		{name: "to find zones, then it returns unauthorized", method: http.MethodGet, path: "/zones", expectedCode: http.StatusUnauthorized},
		{name: "to execute a zone, then it returns unauthorized", method: http.MethodPost, path: "/zones/bbf/execute", expectedCode: http.StatusUnauthorized},
		{name: "to execute a fertilizer zone, then it returns unauthorized", method: http.MethodPost, path: "/zones/fertilizer/bbf/execute", expectedCode: http.StatusUnauthorized},
		{name: "to execute a clean zone, then it returns unauthorized", method: http.MethodPost, path: "/zones/clean/bbf/execute", expectedCode: http.StatusUnauthorized},
		{name: "to update a zone, then it returns unauthorized", method: http.MethodPut, path: "/zones/bbf", expectedCode: http.StatusUnauthorized},
		{name: "to remove a zone, then it returns unauthorized", method: http.MethodDelete, path: "/zones/bbf", expectedCode: http.StatusUnauthorized},
		{name: "to find the status, then it returns unauthorized", method: http.MethodGet, path: "/status", expectedCode: http.StatusUnauthorized},
		{name: "to activate the server, then it returns unauthorized", method: http.MethodPatch, path: "/status/activate", expectedCode: http.StatusUnauthorized},
		{name: "to find the weather, then it returns unauthorized", method: http.MethodGet, path: "/weather", expectedCode: http.StatusUnauthorized},
		{name: "to find all programs, then it returns unauthorized", method: http.MethodGet, path: "/programs", expectedCode: http.StatusUnauthorized},
		{name: "to create a daily program, then it returns unauthorized", method: http.MethodPost, path: "/programs/daily", expectedCode: http.StatusUnauthorized},
		{name: "to remove a daily program, then it returns unauthorized", method: http.MethodDelete, path: "/programs/daily/12:45", expectedCode: http.StatusUnauthorized},
		{name: "to create an odd program, then it returns unauthorized", method: http.MethodPost, path: "/programs/odd", expectedCode: http.StatusUnauthorized},
		{name: "to remove an odd program, then it returns unauthorized", method: http.MethodDelete, path: "/programs/odd/12:45", expectedCode: http.StatusUnauthorized},
		{name: "to create an even program, then it returns unauthorized", method: http.MethodPost, path: "/programs/even", expectedCode: http.StatusUnauthorized},
		{name: "to remove an even program, then it returns unauthorized", method: http.MethodDelete, path: "/programs/even/12:45", expectedCode: http.StatusUnauthorized},
		{name: "to create a weekly program, then it returns unauthorized", method: http.MethodPost, path: "/programs/weekly", expectedCode: http.StatusUnauthorized},
		{name: "to remove a weekly program, then it returns unauthorized", method: http.MethodDelete, path: "/programs/weekly/Monday", expectedCode: http.StatusUnauthorized},
		{name: "to create a temperature program, then it returns unauthorized", method: http.MethodPost, path: "/programs/temperature", expectedCode: http.StatusUnauthorized},
		{name: "to update a temperature program, then it returns unauthorized", method: http.MethodPut, path: "/programs/temperature/20", expectedCode: http.StatusUnauthorized},
		{name: "to remove a temperature program, then it returns unauthorized", method: http.MethodDelete, path: "/programs/temperature/20", expectedCode: http.StatusUnauthorized},
		{name: "to find the execution logs, then it returns unauthorized", method: http.MethodGet, path: "/logs", expectedCode: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(`Given the app running,
		when a request without authorization is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			resp, err := st.Request(t, tt.method, tt.path, nil, false)
			require.NoError(t, err)
			_ = resp.Body.Close()
			require.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}

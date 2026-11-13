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
		{name: "to the homepage", method: http.MethodGet, path: "/", expectedCode: http.StatusOK},
		{name: "to create a zone", method: http.MethodPost, path: "/zones", expectedCode: http.StatusUnauthorized},
		{name: "to find zones", method: http.MethodGet, path: "/zones", expectedCode: http.StatusUnauthorized},
		{name: "to execute a zone", method: http.MethodPost, path: "/zones/bbf/execute", expectedCode: http.StatusUnauthorized},
		{name: "to execute a fertilizer zone", method: http.MethodPost, path: "/zones/fertilizer/bbf/execute", expectedCode: http.StatusUnauthorized},
		{name: "to execute a clean zone", method: http.MethodPost, path: "/zones/clean/bbf/execute", expectedCode: http.StatusUnauthorized},
		{name: "to update a zone", method: http.MethodPut, path: "/zones/bbf", expectedCode: http.StatusUnauthorized},
		{name: "to remove a zone", method: http.MethodDelete, path: "/zones/bbf", expectedCode: http.StatusUnauthorized},
		{name: "to find the status", method: http.MethodGet, path: "/status", expectedCode: http.StatusUnauthorized},
		{name: "to activate the server", method: http.MethodPatch, path: "/status/activate", expectedCode: http.StatusUnauthorized},
		{name: "to find the weather", method: http.MethodGet, path: "/weather", expectedCode: http.StatusUnauthorized},
		{name: "to find all programs", method: http.MethodGet, path: "/programs", expectedCode: http.StatusUnauthorized},
		{name: "to create a daily program", method: http.MethodPost, path: "/programs/daily", expectedCode: http.StatusUnauthorized},
		{name: "to remove a daily program", method: http.MethodDelete, path: "/programs/daily/12:45", expectedCode: http.StatusUnauthorized},
		{name: "to create an odd program", method: http.MethodPost, path: "/programs/odd", expectedCode: http.StatusUnauthorized},
		{name: "to remove an odd program", method: http.MethodDelete, path: "/programs/odd/12:45", expectedCode: http.StatusUnauthorized},
		{name: "to create an even program", method: http.MethodPost, path: "/programs/even", expectedCode: http.StatusUnauthorized},
		{name: "to remove an even program", method: http.MethodDelete, path: "/programs/even/12:45", expectedCode: http.StatusUnauthorized},
		{name: "to create a weekly program", method: http.MethodPost, path: "/programs/weekly", expectedCode: http.StatusUnauthorized},
		{name: "to remove a weekly program", method: http.MethodDelete, path: "/programs/weekly/Monday", expectedCode: http.StatusUnauthorized},
		{name: "to create a temperature program", method: http.MethodPost, path: "/programs/temperature", expectedCode: http.StatusUnauthorized},
		{name: "to update a temperature program", method: http.MethodPut, path: "/programs/temperature/20", expectedCode: http.StatusUnauthorized},
		{name: "to remove a temperature program", method: http.MethodDelete, path: "/programs/temperature/20", expectedCode: http.StatusUnauthorized},
		{name: "to find the execution logs", method: http.MethodGet, path: "/logs", expectedCode: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		outcome := "it returns unauthorized"
		if tt.expectedCode == http.StatusOK {
			outcome = "it returns ok"
		}
		t.Run(`Given the app running,
		when a request without authorization is sent `+tt.name+`, then `+outcome, func(t *testing.T) {
			t.Parallel()
			resp, err := st.Request(t, tt.method, tt.path, nil, false)
			require.NoError(t, err)
			_ = resp.Body.Close()
			require.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}

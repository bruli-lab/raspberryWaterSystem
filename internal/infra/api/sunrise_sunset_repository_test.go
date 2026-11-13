package api_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"github.com/bruli/raspberryWaterSystem/internal/infra/api"
	"github.com/stretchr/testify/require"
)

const validLightBody = `{"results":{"sunrise":"2026-11-13T06:45:00+00:00","sunset":"2026-11-13T16:30:00+00:00"},"status":"OK"}`

type fakeSunriseSunsetAPI struct {
	status   int
	body     string
	requests atomic.Int32
	query    atomic.Value
}

func (f *fakeSunriseSunsetAPI) start(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requests.Add(1)
		f.query.Store(r.URL.Path + "?" + r.URL.RawQuery)
		w.WriteHeader(f.status)
		_, _ = w.Write([]byte(f.body))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func TestSunriseSunsetRepositoryFind(t *testing.T) {
	location, err := time.LoadLocation("Europe/Madrid")
	require.NoError(t, err)
	date := time.Date(2026, time.November, 13, 22, 30, 0, 0, time.UTC)
	expectedLight, err := status.NewLight(
		time.Date(2026, time.November, 13, 6, 45, 0, 0, time.UTC).In(location),
		time.Date(2026, time.November, 13, 16, 30, 0, 0, time.UTC).In(location),
	)
	require.NoError(t, err)
	tests := []struct {
		name          string
		status        int
		body          string
		finds         int
		expectedErr   bool
		expectedLight *status.Light
		expectedCalls int32
	}{
		{
			name:          "and the api returns a non ok status, then it returns an error",
			status:        http.StatusServiceUnavailable,
			finds:         1,
			expectedErr:   true,
			expectedCalls: 1,
		},
		{
			name:          "and the api returns an invalid body, then it returns an error",
			status:        http.StatusOK,
			body:          "not json",
			finds:         1,
			expectedErr:   true,
			expectedCalls: 1,
		},
		{
			name:          "and the api returns an empty sunrise, then it returns an error",
			status:        http.StatusOK,
			body:          `{"results":{"sunset":"2026-11-13T16:30:00+00:00"}}`,
			finds:         1,
			expectedErr:   true,
			expectedCalls: 1,
		},
		{
			name:          "and the api fails twice, then it does not cache the failure and asks again",
			status:        http.StatusInternalServerError,
			finds:         2,
			expectedErr:   true,
			expectedCalls: 2,
		},
		{
			name:          "and the api returns a valid light, then it returns it in the local location",
			status:        http.StatusOK,
			body:          validLightBody,
			finds:         1,
			expectedLight: expectedLight,
			expectedCalls: 1,
		},
		{
			name:          "twice for the same local day, then it asks the api only once",
			status:        http.StatusOK,
			body:          validLightBody,
			finds:         2,
			expectedLight: expectedLight,
			expectedCalls: 1,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a SunriseSunsetRepository,
		when Find method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			fakeAPI := &fakeSunriseSunsetAPI{status: tt.status, body: tt.body}
			repo := api.NewSunriseSunsetRepository(fakeAPI.start(t), 41.38, 2.17, location, time.Second)

			var (
				light   *status.Light
				findErr error
			)
			for range tt.finds {
				light, findErr = repo.Find(t.Context(), date)
			}

			require.Equal(t, tt.expectedCalls, fakeAPI.requests.Load())
			require.Equal(t, "/json?lat=41.38&lng=2.17&date=2026-11-13&formatted=0", fakeAPI.query.Load())
			if tt.expectedErr {
				require.Error(t, findErr)
				return
			}
			require.NoError(t, findErr)
			require.Equal(t, tt.expectedLight, light)
		})
	}
}

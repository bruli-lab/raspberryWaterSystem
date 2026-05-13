//go:build functional

package functional

import (
	"net/http"
	"testing"

	http2 "github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/stretchr/testify/require"
)

func TestWeather(t *testing.T) {
	t.Run(`Given a find weather endpoint,`, func(t *testing.T) {
		t.Run(`when a request without authorization is sent,
		then it returns an unauthorized`, func(t *testing.T) {
			resp, err := buildRequestAndSend(ctx, nil, nil, http.MethodGet, "/weather", cl)
			require.NoError(t, err)
			require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`when a request with authorization is sent,
		then it returns an valid response`, func(t *testing.T) {
			resp, err := buildRequestAndSend(ctx, nil, authorizationHeader(), http.MethodGet, "/weather", cl)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			var schema http2.WeatherResponse
			readResponse(t, resp, &schema)
			require.NotEqual(t, float64(0), schema.Humidity)
			require.NotEqual(t, float64(0), schema.Temperature)
		})
	})
}

//go:build functional

package functional_test

import (
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/bruli/raspberryWaterSystem/tests/harness"
	"github.com/stretchr/testify/require"
)

func TestWeather(t *testing.T) {
	st := harness.Start(t, appBinary)
	tests := []struct {
		name string
	}{
		{name: "then it returns the temperature and humidity of the sensor"},
	}
	for _, tt := range tests {
		t.Run(`Given the app running,
		when the weather is requested `+tt.name, func(t *testing.T) {
			t.Parallel()
			var weather http.WeatherResponse
			st.Get(t, "/weather", &weather)

			require.NotZero(t, weather.Humidity)
			require.NotZero(t, weather.Temperature)
		})
	}
}

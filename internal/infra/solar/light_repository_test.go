package solar_test

import (
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/infra/solar"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestLightRepositoryFind(t *testing.T) {
	madrid, err := time.LoadLocation("Europe/Madrid")
	require.NoError(t, err)
	tests := []struct {
		name                            string
		latitude, longitude             float64
		date                            time.Time
		expectedSunrise, expectedSunset time.Time
		expectedErr                     error
	}{
		{
			name:            "on the summer solstice, then it returns the sunrise and sunset of that day",
			latitude:        41.545,
			longitude:       2.109,
			date:            time.Date(2026, 6, 21, 10, 0, 0, 0, madrid),
			expectedSunrise: time.Date(2026, 6, 21, 4, 17, 40, 0, time.UTC),
			expectedSunset:  time.Date(2026, 6, 21, 19, 29, 6, 0, time.UTC),
		},
		{
			name:            "on the winter solstice, then it returns the sunrise and sunset of that day",
			latitude:        41.545,
			longitude:       2.109,
			date:            time.Date(2026, 12, 21, 23, 30, 0, 0, madrid),
			expectedSunrise: time.Date(2026, 12, 21, 7, 14, 36, 0, time.UTC),
			expectedSunset:  time.Date(2026, 12, 21, 16, 24, 38, 0, time.UTC),
		},
		{
			name:            "with a UTC date that is already the next day in the local time zone, then it returns the light of the local day",
			latitude:        41.545,
			longitude:       2.109,
			date:            time.Date(2026, 12, 20, 23, 30, 0, 0, time.UTC),
			expectedSunrise: time.Date(2026, 12, 21, 7, 14, 36, 0, time.UTC),
			expectedSunset:  time.Date(2026, 12, 21, 16, 24, 38, 0, time.UTC),
		},
		{
			name:        "during the polar night, then it returns a no sunrise error",
			latitude:    80,
			longitude:   15,
			date:        time.Date(2026, 12, 21, 12, 0, 0, 0, time.UTC),
			expectedErr: solar.NoSunriseError{},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a solar LightRepository,
		when Find method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := solar.NewLightRepository(tt.latitude, tt.longitude, madrid, noop.NewTracerProvider().Tracer("test"))

			light, err := repo.Find(t.Context(), tt.date)
			if tt.expectedErr != nil {
				require.IsType(t, tt.expectedErr, err)
				return
			}
			require.NoError(t, err)
			require.False(t, light.IsDay(tt.expectedSunrise.Add(-2*time.Minute)))
			require.True(t, light.IsDay(tt.expectedSunrise.Add(2*time.Minute)))
			require.True(t, light.IsDay(tt.expectedSunset.Add(-2*time.Minute)))
			require.False(t, light.IsDay(tt.expectedSunset.Add(2*time.Minute)))
		})
	}
}

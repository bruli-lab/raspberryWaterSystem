package status_test

import (
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run(`Given a Status struct,
	when New function is called, 
	then it returns a valid active status`, func(t *testing.T) {
		t.Parallel()
		start := time.Now()
		weather := fixtures.WeatherBuilder{}.Build()
		light := fixtures.LightBuilder{}.Build()
		st := status.New(start, weather, light)
		require.Equal(t, start, st.SystemStartedAt())
		require.Equal(t, weather, st.Weather())
		require.True(t, st.IsActive())

		st.Deactivate()
		require.False(t, st.IsActive())
	})
}

package zone_test

import (
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestZoneIgnoreWhileCleaning(t *testing.T) {
	t.Run(`Given a Zone,
	when IgnoreWhileCleaning method is called,
	then it records an ignored event with the cleaning reason`, func(t *testing.T) {
		t.Parallel()
		zo := fixtures.ZoneBuilder{}.Build()

		zo.IgnoreWhileCleaning()

		events := zo.Events()
		require.Len(t, events, 1)
		ignored, ok := events[0].(zone.Ignored)
		require.True(t, ok)
		require.Equal(t, zone.CleaningIgnoredReason, ignored.Reason)
		require.Equal(t, zo.Name(), ignored.ZoneName)
	})
}

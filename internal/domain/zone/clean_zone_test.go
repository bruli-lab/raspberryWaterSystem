package zone_test

import (
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestCleanZone_Execute(t *testing.T) {
	t.Run(`Given a CleanZone struct,
	when Execute method is called,
	then it returns a CleanZoneExecuted event`, func(t *testing.T) {
		zo := fixtures.ZoneBuilder{}.Build()
		clean := zone.NewCleanZone(zo)
		clean.Execute()
		events := clean.Events()
		require.Len(t, events, 1)
		ev := events[0]
		event, ok := ev.(zone.CleanZoneExecuted)
		require.True(t, ok)
		require.Equal(t, zo.Name(), event.ZoneName)
		require.Equal(t, zo.Id(), event.ZoneID)
		require.Equal(t, uint(zone.DefaultCleanZone.Seconds()), event.ZoneSeconds)
		require.Equal(t, uint(zo.StabilizationFlux().Seconds()), event.StabilizationZoneSeconds)
		require.Equal(t, zone.DefaultStabilizationFlux, zo.StabilizationFlux())
		require.Equal(t, zone.DefaultWaitCleanZone, event.WaitDuration)
		require.Equal(t, zone.DefaultClarifyCleanZone, event.ClarifyDuration)
	})
}

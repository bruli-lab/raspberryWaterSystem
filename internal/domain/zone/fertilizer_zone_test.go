package zone_test

import (
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestFertilizerZone_Execute(t *testing.T) {
	type args struct {
		seconds uint
	}
	tests := []struct {
		name        string
		expectedErr error
		args        args
	}{
		{
			name:        "with seconds exceeded limit, then it returns an invalid seconds error",
			args:        args{seconds: 121},
			expectedErr: program.InvalidSecondsError{},
		},
		{
			name:        "with zero seconds, then it returns an invalid seconds error",
			args:        args{seconds: 0},
			expectedErr: program.InvalidSecondsError{},
		},
		{
			name: "with valid seconds, then it returns a valid event",
			args: args{seconds: 20},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a FertilizerZone built struct,
		when Execute method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			zo := fixtures.ZoneBuilder{}.Build()
			fz := zone.NewFertilizerZone(zo)
			err := fz.Execute(tt.args.seconds)
			if tt.expectedErr != nil {
				require.ErrorAs(t, err, &program.InvalidSecondsError{})
				return
			}
			require.NoError(t, err)
			ev := fz.Events()
			require.Len(t, ev, 1)
			event, ok := ev[0].(zone.FertilizerZoneExecuted)
			zoneRelays := make([]string, len(zo.Relays()))
			for i, r := range zo.Relays() {
				zoneRelays[i] = r.Pin()
			}
			cleanUPPIn, err := zone.ParseRelay(zone.CleanValveID)
			require.NoError(t, err)
			fertilizerPIn, err := zone.ParseRelay(zone.FertilizerPumpID)
			require.NoError(t, err)
			airZonePin, err := zone.ParseRelay(zone.AirRelayID)
			require.NoError(t, err)
			fertValvePin, err := zone.ParseRelay(zone.FertilizerValveID)
			require.NoError(t, err)

			require.True(t, ok)
			require.Equal(t, zo.Name(), event.ZoneName)
			require.Equal(t, zo.Id(), event.ZoneID)
			require.Equal(t, tt.args.seconds, event.ZoneSeconds)
			require.Equal(t, zo.StabilizationFlux().Seconds(), float64(event.StabilizationZoneSeconds))
			require.Equal(t, zoneRelays, event.ZoneRelayPins)
			require.Equal(t, cleanUPPIn.Pin(), event.CleanValveRelayPin)
			require.Equal(t, uint(zone.CleanValveDefaultTime.Seconds()), event.CleanValveSeconds)
			require.Equal(t, tt.args.seconds+uint(zone.CleanValveDefaultTime.Seconds()), event.FertilizerPumpSeconds)
			require.Equal(t, fertilizerPIn.Pin(), event.FertilizerPumpRelayPin)
			require.Equal(t, uint(zone.AirZoneDefaultTime.Seconds()), event.AirZoneSeconds)
			require.Equal(t, airZonePin.Pin(), event.AirZoneRelayPin)
			require.Equal(t, tt.args.seconds, event.FertilizerValveSeconds)
			require.Equal(t, fertValvePin.Pin(), event.FertilizerValveRelayPin)
		})
	}
}

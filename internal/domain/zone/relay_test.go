package zone_test

import (
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/stretchr/testify/require"
)

func TestParseRelay(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		expectedRelay zone.RelayID
	}{
		{
			name: "with an unknown id, then it return a unknown relay error",
			id:   99,
		},
		{
			name:          "with an unknown id, then it return a unknown valid relay id",
			id:            1,
			expectedRelay: zone.OneRelayID,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a ParseRelay function,
		when is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			relay, err := zone.ParseRelay(tt.id)
			if err != nil {
				require.ErrorIs(t, err, zone.ErrUnknownRelay)
				return
			}
			require.Equal(t, tt.expectedRelay, relay.Id())
		})
	}
}

func TestFixedRelays(t *testing.T) {
	tests := []struct {
		name  string
		relay zone.Relay
		id    int
	}{
		{name: "air zone", relay: zone.AirZone{}.Relay(), id: zone.AirRelayID},
		{name: "clean valve", relay: zone.CleanValveZone{}.Relay(), id: zone.CleanValveID},
		{name: "fertilizer pump", relay: zone.FertilizerPumpZone{}.Relay(), id: zone.FertilizerPumpID},
		{name: "fertilizer valve", relay: zone.FertilizerValveZone{}.Relay(), id: zone.FertilizerValveID},
	}
	for _, tt := range tests {
		t.Run(`Given the `+tt.name+` fixed relay, when Relay method is called, then it returns the enabled relay with its pin`, func(t *testing.T) {
			t.Parallel()
			expected, err := zone.ParseRelay(tt.id)
			require.NoError(t, err)
			require.Equal(t, expected, tt.relay)
			require.NotEmpty(t, tt.relay.Pin())
		})
	}
}

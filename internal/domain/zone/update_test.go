package zone_test

import (
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestZoneUpdate(t *testing.T) {
	relay, err := zone.ParseRelay(2)
	require.NoError(t, err)
	tests := []struct {
		name, zoneName string
		relays         []zone.Relay
		expectedErr    error
	}{
		{
			name:        "with an empty name, then it returns an invalid zone name error",
			relays:      []zone.Relay{relay},
			expectedErr: zone.ErrInvalidZoneName,
		},
		{
			name:        "without relays, then it returns an invalid zone relays error",
			zoneName:    "new name",
			expectedErr: zone.ErrInvalidZoneRelays,
		},
		{
			name:     "with valid values, then it changes name and relays keeping the id",
			zoneName: "new name",
			relays:   []zone.Relay{relay},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a Zone, when Update method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			zo := fixtures.ZoneBuilder{}.Build()
			id, name, relays := zo.Id(), zo.Name(), zo.Relays()

			err := zo.Update(tt.zoneName, tt.relays)

			require.Equal(t, id, zo.Id())
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Equal(t, name, zo.Name())
				require.Equal(t, relays, zo.Relays())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.zoneName, zo.Name())
			require.Equal(t, tt.relays, zo.Relays())
		})
	}
}

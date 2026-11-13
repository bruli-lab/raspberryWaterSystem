package listener_test

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/listener"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func fertilizerZoneEvent() zone.FertilizerZoneExecuted {
	return zone.FertilizerZoneExecuted{
		BasicEvent:               event.NewBasicEvent(zone.FertilizerZoneExecutedEventName, uuid.New(), uuid.NewString()),
		ZoneID:                   "bbf",
		ZoneName:                 "bonsai big with fertilizer",
		ZoneSeconds:              10,
		StabilizationZoneSeconds: 3,
		ZoneRelayPins:            []string{"1", "2"},
		CleanValveSeconds:        15,
		CleanValveRelayPin:       "3",
		FertilizerPumpSeconds:    25,
		FertilizerPumpRelayPin:   "4",
		AirZoneSeconds:           15,
		AirZoneRelayPin:          "5",
		FertilizerValveSeconds:   10,
		FertilizerValveRelayPin:  "6",
	}
}

func TestExecutePinsOnExecuteFertilizerZoneListen(t *testing.T) {
	errAir := errors.New("air relay failed")
	tests := []struct {
		name             string
		failingPins      map[string]error
		expectedErr      error
		expectedPins     []app.ExecutePinsCmd
		expectedWaits    []time.Duration
		expectsRecording bool
	}{
		{
			name: "and every relay works, then it runs air first, waits the stabilization, runs zone, valve, clean and pump, and records the irrigation",
			expectedPins: []app.ExecutePinsCmd{
				{Seconds: 15, Pins: []string{"5"}},
				{Seconds: 28, Pins: []string{"1", "2"}},
				{Seconds: 10, Pins: []string{"6"}},
				{Seconds: 15, Pins: []string{"3"}},
				{Seconds: 25, Pins: []string{"4"}},
			},
			expectedWaits:    []time.Duration{3 * time.Second},
			expectsRecording: true,
		},
		{
			name:         "and the air relay fails, then it returns that error without opening the zone nor recording the irrigation",
			failingPins:  map[string]error{"5": errAir},
			expectedErr:  errAir,
			expectedPins: []app.ExecutePinsCmd{{Seconds: 15, Pins: []string{"5"}}},
		},
	}
	for _, tt := range tests {
		t.Run(`Given an ExecutePinsOnExecuteFertilizerZone listener, when Listen method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			recorder := newPinsRecorder()
			recorder.failingPins = tt.failingPins
			list := listener.NewExecutePinsOnExecuteFertilizerZone(recorder.commandHandler(), recorder.wait, tracer(), slog.New(slog.DiscardHandler))

			err := list.Listen(t.Context(), fertilizerZoneEvent())

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
			pins := recorder.recordedPins()
			require.ElementsMatch(t, tt.expectedPins, pins)
			require.Equal(t, tt.expectedPins[0], pins[0])
			require.Equal(t, tt.expectedWaits, recorder.waits)
			if !tt.expectsRecording {
				require.Zero(t, recorder.logs)
				require.Empty(t, recorder.messages)
				return
			}
			require.Less(t, indexOfPin(pins, "6"), indexOfPin(pins, "3"))
			require.Equal(t, 1, recorder.logs)
			require.Equal(t, []string{"bonsai big with fertilizer fertilizer zone executed during 10s"}, recorder.messages)
		})
	}
}

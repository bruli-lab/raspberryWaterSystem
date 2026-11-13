package listener_test

import (
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

func TestExecutePinsOnExecuteCleanZoneCycle(t *testing.T) {
	tests := []struct {
		name          string
		expectedPins  []app.ExecutePinsCmd
		expectedWaits []time.Duration
	}{
		{
			name: "then it cleans with the product, waits, clarifies, notifies twice and leaves the clean mode",
			expectedPins: []app.ExecutePinsCmd{
				{Seconds: 13, Pins: []string{"1", "2"}, CleanCycle: true},
				{Seconds: 10, Pins: []string{"6"}, CleanCycle: true},
				{Seconds: 10, Pins: []string{"4"}, CleanCycle: true},
				{Seconds: 18, Pins: []string{"1", "2"}, CleanCycle: true},
				{Seconds: 15, Pins: []string{"4"}, CleanCycle: true},
				{Seconds: 15, Pins: []string{"6"}, CleanCycle: true},
			},
			expectedWaits: []time.Duration{3 * time.Second, 20 * time.Minute, 3 * time.Second},
		},
	}
	for _, tt := range tests {
		t.Run(`Given an ExecutePinsOnExecuteCleanZone listener, when Listen method is called with a clean zone event `+tt.name, func(t *testing.T) {
			t.Parallel()
			recorder := newPinsRecorder()
			list := listener.NewExecutePinsOnExecuteCleanZone(recorder.commandHandler(), recorder.wait, tracer(), slog.New(slog.DiscardHandler))

			err := list.Listen(t.Context(), zone.CleanZoneExecuted{
				BasicEvent:               event.NewBasicEvent(zone.CleanZoneExecutedEventName, uuid.New(), uuid.NewString()),
				ZoneID:                   "bbf",
				ZoneName:                 "bonsai big with fertilizer",
				ZoneSeconds:              10,
				StabilizationZoneSeconds: 3,
				ZoneRelayPins:            []string{"1", "2"},
				FertilizerPumpSeconds:    10,
				FertilizerPumpRelayPin:   "4",
				FertilizerValveSeconds:   10,
				FertilizerValveRelayPin:  "6",
				WaitDuration:             20 * time.Minute,
				ClarifyDuration:          15 * time.Second,
			})
			require.NoError(t, err)
			select {
			case <-recorder.cleanChanged:
			case <-time.After(5 * time.Second):
				t.Fatal("clean cycle did not finish")
			}

			pins := recorder.recordedPins()
			require.Len(t, pins, len(tt.expectedPins))
			require.ElementsMatch(t, tt.expectedPins[:3], pins[:3])
			require.ElementsMatch(t, tt.expectedPins[3:], pins[3:])
			require.Equal(t, tt.expectedWaits, recorder.waits)
			require.Equal(t, []string{
				"bonsai big with fertilizer clean zone executed during 10s",
				"clarify zone bonsai big with fertilizer finished!!",
			}, recorder.messages)
			require.Equal(t, []bool{false}, recorder.cleanStates)
		})
	}
}

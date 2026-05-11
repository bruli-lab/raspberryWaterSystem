package zone

import (
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/google/uuid"
)

const (
	DefaultCleanZone        = 2 * time.Minute
	DefaultWaitCleanZone    = 15 * time.Minute
	DefaultClarifyCleanZone = 2 * time.Minute
)

type CleanZone struct {
	cqs.BasicAggregateRoot
	zone              *Zone
	fertilizerPump    *FertilizerPumpZone
	fertilizerValvule *FertilizerValvuleZone
}

func (z *CleanZone) Execute() {
	pins := make([]string, len(z.zone.Relays()))
	for i, p := range z.zone.Relays() {
		pins[i] = p.pin
	}
	seconds := uint(DefaultCleanZone.Seconds())
	z.Record(CleanZoneExecuted{
		BasicEvent:                cqs.NewBasicEvent(CleanZoneExecutedEventName, uuid.New(), z.zone.Id()),
		ZoneID:                    z.zone.Id(),
		ZoneName:                  z.zone.Name(),
		ZoneSeconds:               seconds,
		StabilizationZoneSeconds:  uint(z.zone.StabilizationFlux().Seconds()),
		ZoneRelayPins:             pins,
		FertilizerPumpSeconds:     seconds,
		FertilizerPumpRelayPin:    z.fertilizerPump.Relay().Pin(),
		FertilizerValvuleSeconds:  seconds,
		FertilizerValvuleRelayPin: z.fertilizerValvule.Relay().Pin(),
		WaitDuration:              DefaultWaitCleanZone,
		ClarifyDuration:           DefaultClarifyCleanZone,
	})
}

func NewCleanZone(zone *Zone) *CleanZone {
	return &CleanZone{
		BasicAggregateRoot: cqs.NewBasicAggregateRoot(),
		zone:               zone,
		fertilizerPump:     &FertilizerPumpZone{},
		fertilizerValvule:  &FertilizerValvuleZone{},
	}
}

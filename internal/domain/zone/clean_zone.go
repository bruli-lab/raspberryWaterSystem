package zone

import (
	"time"

	"github.com/bruli-lab/go-core/event"
	"github.com/google/uuid"
)

const (
	DefaultCleanZone        = 2 * time.Minute
	DefaultWaitCleanZone    = 15 * time.Minute
	DefaultClarifyCleanZone = 2 * time.Minute
)

type CleanZone struct {
	event.BasicAggregateRoot
	zone            *Zone
	fertilizerPump  *FertilizerPumpZone
	fertilizerValve *FertilizerValveZone
}

func (z *CleanZone) Execute() {
	pins := make([]string, len(z.zone.Relays()))
	for i, p := range z.zone.Relays() {
		pins[i] = p.pin
	}
	seconds := uint(DefaultCleanZone.Seconds())
	z.Record(CleanZoneExecuted{
		BasicEvent:               event.NewBasicEvent(CleanZoneExecutedEventName, uuid.New(), z.zone.Id()),
		ZoneID:                   z.zone.Id(),
		ZoneName:                 z.zone.Name(),
		ZoneSeconds:              seconds,
		StabilizationZoneSeconds: uint(z.zone.StabilizationFlux().Seconds()),
		ZoneRelayPins:            pins,
		FertilizerPumpSeconds:    seconds,
		FertilizerPumpRelayPin:   z.fertilizerPump.Relay().Pin(),
		FertilizerValveSeconds:   seconds,
		FertilizerValveRelayPin:  z.fertilizerValve.Relay().Pin(),
		WaitDuration:             DefaultWaitCleanZone,
		ClarifyDuration:          DefaultClarifyCleanZone,
	})
}

func NewCleanZone(zone *Zone) *CleanZone {
	return &CleanZone{
		BasicAggregateRoot: event.NewBasicAggregateRoot(),
		zone:               zone,
		fertilizerPump:     &FertilizerPumpZone{},
		fertilizerValve:    &FertilizerValveZone{},
	}
}

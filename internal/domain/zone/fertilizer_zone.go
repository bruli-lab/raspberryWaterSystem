package zone

import (
	"time"

	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/google/uuid"
)

const (
	AirZoneDefaultTime    = 15 * time.Second
	CleanValveDefaultTime = 15 * time.Second
)

type FertilizerZone struct {
	event.BasicAggregateRoot
	zone            *Zone
	airZone         *AirZone
	fertilizerPump  *FertilizerPumpZone
	cleanValve      *CleanValveZone
	fertilizerValve *FertilizerValveZone
}

func (z *FertilizerZone) Execute(seconds uint) error {
	if _, err := program.ParseSeconds(int(seconds)); err != nil {
		return err
	}
	pins := make([]string, len(z.zone.Relays()))
	for i, p := range z.zone.Relays() {
		pins[i] = p.pin
	}
	z.Record(FertilizerZoneExecuted{
		BasicEvent:               event.NewBasicEvent(FertilizerZoneExecutedEventName, uuid.New(), z.zone.Id()),
		ZoneID:                   z.zone.Id(),
		ZoneName:                 z.zone.Name(),
		ZoneSeconds:              seconds,
		StabilizationZoneSeconds: uint(z.zone.StabilizationFlux().Seconds()),
		ZoneRelayPins:            pins,
		CleanValveSeconds:        uint(z.cleanValve.Seconds().Seconds()),
		CleanValveRelayPin:       z.cleanValve.Relay().Pin(),
		FertilizerPumpSeconds:    seconds + uint(z.cleanValve.Seconds().Seconds()),
		FertilizerPumpRelayPin:   z.fertilizerPump.Relay().Pin(),
		AirZoneSeconds:           uint(z.airZone.Seconds().Seconds()),
		AirZoneRelayPin:          z.airZone.Relay().Pin(),
		FertilizerValveSeconds:   seconds,
		FertilizerValveRelayPin:  z.fertilizerValve.Relay().Pin(),
	})
	return nil
}

func (z *FertilizerZone) ExecuteWithStatus(active, raining bool, seconds uint) error {
	if ignored, isIgnored := ignoredByStatus(z.zone.id, z.zone.name, active, raining); isIgnored {
		z.Record(ignored)
		return nil
	}
	return z.Execute(seconds)
}

func NewFertilizerZone(zone *Zone) *FertilizerZone {
	return &FertilizerZone{
		BasicAggregateRoot: event.NewBasicAggregateRoot(),
		zone:               zone,
		airZone:            &AirZone{},
		fertilizerPump:     &FertilizerPumpZone{},
		cleanValve:         &CleanValveZone{},
		fertilizerValve:    &FertilizerValveZone{},
	}
}

type FertilizerValveZone struct{}

func (f FertilizerValveZone) Relay() Relay {
	return fertilizerValveRelay
}

type CleanValveZone struct{}

func (c CleanValveZone) Seconds() time.Duration {
	return CleanValveDefaultTime
}

func (c CleanValveZone) Relay() Relay {
	return cleanValveRelay
}

type FertilizerPumpZone struct{}

func (f FertilizerPumpZone) Relay() Relay {
	return fertilizerPumpRelay
}

type AirZone struct{}

func (a AirZone) Seconds() time.Duration {
	return AirZoneDefaultTime
}

func (a AirZone) Relay() Relay {
	return airRelay
}

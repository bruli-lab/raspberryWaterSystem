package zone

import (
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/google/uuid"
)

const (
	AirZoneDefaultTime      = 15 * time.Second
	CleanValvuleDefaultTime = 15 * time.Second
)

type FertilizerZone struct {
	cqs.BasicAggregateRoot
	zone              *Zone
	airZone           *AireZone
	fertilizerPump    *FertilizerPumpZone
	cleanValvule      *CleanValvuleZone
	fertilizerValvule *FertilizerValvuleZone
}

func (z *FertilizerZone) Execute(seconds uint) error {
	if seconds > ExecutionSecondsLimit {
		return ErrInvalidSecondsExecutionZone
	}
	pins := make([]string, len(z.zone.Relays()))
	for i, p := range z.zone.Relays() {
		pins[i] = p.pin
	}
	z.Record(FertilizerZoneExecuted{
		BasicEvent:                cqs.NewBasicEvent(FertilizerZoneExecutedEventName, uuid.New(), z.zone.Id()),
		ZoneID:                    z.zone.Id(),
		ZoneName:                  z.zone.Name(),
		ZoneSeconds:               seconds,
		StabilizationZoneSeconds:  uint(z.zone.StabilizationFlux().Seconds()),
		ZoneRelayPins:             pins,
		CleanValvuleSeconds:       uint(z.cleanValvule.Seconds().Seconds()),
		CleanValvuleRelayPin:      z.cleanValvule.Relay().Pin(),
		FertilizerPumpSeconds:     seconds + uint(z.cleanValvule.Seconds().Seconds()),
		FertilizerPumpRelayPin:    z.fertilizerPump.Relay().Pin(),
		AirZoneSeconds:            uint(z.airZone.Seconds().Seconds()),
		AirZoneRelayPin:           z.airZone.Relay().Pin(),
		FertilizerValvuleSeconds:  seconds,
		FertilizerValvuleRelayPin: z.fertilizerValvule.Relay().Pin(),
	})
	return nil
}

func NewFertilizerZone(zone *Zone) *FertilizerZone {
	return &FertilizerZone{
		BasicAggregateRoot: cqs.NewBasicAggregateRoot(),
		zone:               zone,
		airZone:            &AireZone{},
		fertilizerPump:     &FertilizerPumpZone{},
		cleanValvule:       &CleanValvuleZone{},
		fertilizerValvule:  &FertilizerValvuleZone{},
	}
}

type FertilizerValvuleZone struct{}

func (f FertilizerValvuleZone) Relay() *Relay {
	r, _ := ParseRelay(FertilizerValvuleID)
	return &r
}

type CleanValvuleZone struct{}

func (c CleanValvuleZone) Seconds() time.Duration {
	return CleanValvuleDefaultTime
}

func (c CleanValvuleZone) Relay() *Relay {
	r, _ := ParseRelay(CleanValvuleID)
	return &r
}

type FertilizerPumpZone struct{}

func (f FertilizerPumpZone) Relay() *Relay {
	r, _ := ParseRelay(FertilizerPumpID)
	return &r
}

type AireZone struct{}

func (a AireZone) Seconds() time.Duration {
	return AirZoneDefaultTime
}

func (a AireZone) Relay() *Relay {
	r, _ := ParseRelay(AirRelayID)
	return &r
}

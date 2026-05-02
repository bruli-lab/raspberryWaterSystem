package zone

import (
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/google/uuid"
)

const (
	AirZoneDefaultTime   = 15 * time.Second
	CleanPumpDefaultTime = 15 * time.Second
)

type FertilizerZone struct {
	cqs.BasicAggregateRoot
	zone           *Zone
	airZone        *AireZone
	fertilizerPump *FertilizerPumpZone
	CleanPump      *CleanPumpZone
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
		BasicEvent:               cqs.NewBasicEvent(FertilizerZoneExecutedEventName, uuid.New(), z.zone.Id()),
		ZoneID:                   z.zone.Id(),
		ZoneName:                 z.zone.Name(),
		ZoneSeconds:              seconds,
		StabilizationZoneSeconds: uint(z.zone.StabilizationFlux().Seconds()),
		ZoneRelayPins:            pins,
		CleanPumpSeconds:         uint(z.CleanPump.Seconds().Seconds()),
		CleanPumpRelayPin:        z.CleanPump.Relay().Pin(),
		FertilizerPumpSeconds:    seconds,
		FertilizerPumpRelayPin:   z.fertilizerPump.Relay().Pin(),
		AirZoneSeconds:           uint(z.airZone.Seconds().Seconds()),
		AirZoneRelayPin:          z.airZone.Relay().Pin(),
	})
	return nil
}

func NewFertilizerZone(zone *Zone) *FertilizerZone {
	return &FertilizerZone{
		BasicAggregateRoot: cqs.NewBasicAggregateRoot(),
		zone:               zone,
		airZone:            &AireZone{},
		fertilizerPump:     &FertilizerPumpZone{},
		CleanPump:          &CleanPumpZone{},
	}
}

type CleanPumpZone struct{}

func (c CleanPumpZone) Seconds() time.Duration {
	return CleanPumpDefaultTime
}

func (c CleanPumpZone) Relay() *Relay {
	r, _ := ParseRelay(CleanPumpID)
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

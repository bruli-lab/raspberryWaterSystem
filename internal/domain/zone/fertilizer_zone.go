package zone

import (
	"sync"
	"time"

	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/google/uuid"
)

const (
	AirZoneDefaultTime      = 15 * time.Second
	CleanValvuleDefaultTime = 15 * time.Second
)

type FertilizerZone struct {
	event.BasicAggregateRoot
	zone              *Zone
	airZone           *AireZone
	fertilizerPump    *FertilizerPumpZone
	cleanValvule      *CleanValvuleZone
	fertilizerValvule *FertilizerValvuleZone
	mutex             sync.Mutex
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
		BasicEvent:                event.NewBasicEvent(FertilizerZoneExecutedEventName, uuid.New(), z.zone.Id()),
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

func (z *FertilizerZone) ExecuteWithStatus(active, raining bool, seconds uint) error {
	z.mutex.Lock()
	defer z.mutex.Unlock()
	ignored := Ignored{
		BasicEvent: event.NewBasicEvent(IgnoredEventName, uuid.New(), z.zone.id),
		ZoneName:   z.zone.name,
	}
	switch {
	case !active:
		ignored.Reason = DeactivatedIgnoredReason
		z.Record(ignored)
		return nil
	case raining:
		ignored.Reason = RainingIgnoredReason
		z.Record(ignored)
		return nil
	default:
		return z.Execute(seconds)
	}
}

func NewFertilizerZone(zone *Zone) *FertilizerZone {
	return &FertilizerZone{
		BasicAggregateRoot: event.NewBasicAggregateRoot(),
		zone:               zone,
		airZone:            &AireZone{},
		fertilizerPump:     &FertilizerPumpZone{},
		cleanValvule:       &CleanValvuleZone{},
		fertilizerValvule:  &FertilizerValvuleZone{},
		mutex:              sync.Mutex{},
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

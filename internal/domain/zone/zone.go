package zone

import (
	"errors"
	"sync"
	"time"

	"github.com/bruli-lab/go-core/event"
	"github.com/google/uuid"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
)

const (
	RainingIgnoredReason     = "It's raining!!"
	DeactivatedIgnoredReason = "Deactivated!!"
	CleaningIgnoredReason    = "Cleaning!!"

	DefaultStabilizationFlux = 3 * time.Second
)

var (
	ErrInvalidZoneID     = errors.New("zone id can not be empty")
	ErrInvalidZoneName   = errors.New("zone name can not be empty")
	ErrInvalidZoneRelays = errors.New("zone relays can not be empty")
)

type Zone struct {
	event.BasicAggregateRoot
	id, name string
	relays   []Relay
	mutex    sync.RWMutex
}

func (z *Zone) Id() string {
	return z.id
}

func (z *Zone) Name() string {
	return z.name
}

func (z *Zone) Relays() []Relay {
	return z.relays
}

func (z *Zone) StabilizationFlux() time.Duration {
	return DefaultStabilizationFlux
}

func New(id, name string, relays []Relay) (*Zone, error) {
	z := Zone{
		BasicAggregateRoot: event.NewBasicAggregateRoot(),
		id:                 id,
		name:               name,
		relays:             relays,
		mutex:              sync.RWMutex{},
	}
	if err := z.validate(); err != nil {
		return nil, err
	}
	return &z, nil
}

func HydrateZone(id, name string, relays []Relay) *Zone {
	return &Zone{
		id:     id,
		name:   name,
		relays: relays,
		mutex:  sync.RWMutex{},
	}
}

func (z *Zone) validate() error {
	if z.id == "" {
		return ErrInvalidZoneID
	}
	if z.name == "" {
		return ErrInvalidZoneName
	}
	if len(z.relays) == 0 {
		return ErrInvalidZoneRelays
	}
	return nil
}

func (z *Zone) Execute(seconds uint) error {
	if _, err := program.ParseSeconds(int(seconds)); err != nil {
		return err
	}
	pins := make([]string, len(z.relays))
	for i, p := range z.relays {
		pins[i] = p.pin
	}
	z.Record(Executed{
		BasicEvent:           event.NewBasicEvent(ExecutedEventName, uuid.New(), z.id),
		ZoneID:               z.id,
		ZoneName:             z.name,
		Seconds:              seconds,
		StabilizationSeconds: uint(z.StabilizationFlux().Seconds()),
		RelayPins:            pins,
	})
	return nil
}

func (z *Zone) IgnoreWhileCleaning() {
	z.mutex.Lock()
	defer z.mutex.Unlock()
	z.Record(Ignored{
		BasicEvent: event.NewBasicEvent(IgnoredEventName, uuid.New(), z.id),
		ZoneName:   z.name,
		Reason:     CleaningIgnoredReason,
	})
}

func (z *Zone) ExecuteWithStatus(active, raining bool, seconds uint) error {
	z.mutex.Lock()
	defer z.mutex.Unlock()
	ignored := Ignored{
		BasicEvent: event.NewBasicEvent(IgnoredEventName, uuid.New(), z.id),
		ZoneName:   z.name,
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

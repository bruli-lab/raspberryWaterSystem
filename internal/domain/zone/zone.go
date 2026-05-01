package zone

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
)

const (
	ExecutionSecondsLimit = 300

	RainingIgnoredReason     = "It's raining!!"
	DeactivatedIgnoredReason = "Deactivated!!"

	DefaultStabilizationFlux = 3 * time.Second
)

var (
	ErrInvalidZoneID               = errors.New("zone id can not be empty")
	ErrInvalidZoneName             = errors.New("zone name can not be empty")
	ErrInvalidZoneRelays           = errors.New("zone relays can not be empty")
	ErrInvalidSecondsExecutionZone = errors.New("execution zone has limit 300")
)

type Zone struct {
	cqs.BasicAggregateRoot
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
		BasicAggregateRoot: cqs.NewBasicAggregateRoot(),
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

func (z *Zone) Hydrate(id, name string, relays []Relay) {
	z.id = id
	z.name = name
	z.relays = relays
	z.mutex = sync.RWMutex{}
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
	if seconds > ExecutionSecondsLimit {
		return ErrInvalidSecondsExecutionZone
	}
	pins := make([]string, len(z.relays))
	for i, p := range z.relays {
		pins[i] = p.pin
	}
	z.Record(Executed{
		BasicEvent:           cqs.NewBasicEvent(ExecutedEventName, uuid.New(), z.id),
		ZoneID:               z.id,
		ZoneName:             z.name,
		Seconds:              seconds,
		StabilizationSeconds: uint(z.StabilizationFlux().Seconds()),
		RelayPins:            pins,
	})
	return nil
}

func (z *Zone) ExecuteWithStatus(active, raining bool, seconds uint) error {
	z.mutex.Lock()
	defer z.mutex.Unlock()
	event := Ignored{
		BasicEvent: cqs.NewBasicEvent(IgnoredEventName, uuid.New(), z.id),
		ZoneName:   z.name,
	}
	switch {
	case !active:
		event.Reason = DeactivatedIgnoredReason
		z.Record(event)
		return nil
	case raining:
		event.Reason = RainingIgnoredReason
		z.Record(event)
		return nil
	default:
		return z.Execute(seconds)
	}
}

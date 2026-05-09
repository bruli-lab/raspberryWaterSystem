package zone

import "errors"

const (
	MasterRelayID = iota + 1
	FertilizerPumpRelayID
	BigRelayID
	SmallRelayID
	AirRelayID
	FertilizerPumpID
	CleanValvuleID
	FertilizerValvuleID
)

type RelayID int

func (i RelayID) Int() int {
	return int(i)
}

var ErrUnknownRelay = errors.New("unknown relay")

var enabledRelays = map[RelayID]string{
	MasterRelayID:         "master",
	FertilizerPumpRelayID: "fertilizerPump",
	BigRelayID:            "big",
	SmallRelayID:          "small",
	AirRelayID:            "air",
	CleanValvuleID:        "clean",
	FertilizerPumpID:      "22",
	FertilizerValvuleID:   "fertilizerValvule",
}

type Relay struct {
	id  RelayID
	pin string
}

func (r Relay) Id() RelayID {
	return r.id
}

func (r Relay) Pin() string {
	return r.pin
}

func newRelay(id RelayID) (Relay, error) {
	pin, ok := enabledRelays[id]
	if !ok {
		return Relay{}, ErrUnknownRelay
	}
	return Relay{id: id, pin: pin}, nil
}

func ParseRelay(i int) (Relay, error) {
	return newRelay(RelayID(i))
}

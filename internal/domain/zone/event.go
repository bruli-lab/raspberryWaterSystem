package zone

import (
	"time"

	"github.com/bruli-lab/go-core/event"
)

const (
	ExecutedEventName               = "v1.zone.executed"
	IgnoredEventName                = "v1.zone.ignored"
	FertilizerZoneExecutedEventName = "v1.fertilizer.zone.executed"
	CleanZoneExecutedEventName      = "v1.clean.zone.executed"
)

type Executed struct {
	event.BasicEvent
	ZoneID               string
	ZoneName             string
	Seconds              uint
	StabilizationSeconds uint
	RelayPins            []string
}

type Ignored struct {
	event.BasicEvent
	ZoneName string
	Reason   string
}

type FertilizerZoneExecuted struct {
	event.BasicEvent
	ZoneID                    string
	ZoneName                  string
	ZoneSeconds               uint
	StabilizationZoneSeconds  uint
	ZoneRelayPins             []string
	CleanValvuleSeconds       uint
	CleanValvuleRelayPin      string
	FertilizerPumpSeconds     uint
	FertilizerPumpRelayPin    string
	AirZoneSeconds            uint
	AirZoneRelayPin           string
	FertilizerValvuleSeconds  uint
	FertilizerValvuleRelayPin string
}
type CleanZoneExecuted struct {
	event.BasicEvent
	ZoneID                    string
	ZoneName                  string
	ZoneSeconds               uint
	StabilizationZoneSeconds  uint
	ZoneRelayPins             []string
	FertilizerPumpSeconds     uint
	FertilizerPumpRelayPin    string
	FertilizerValvuleSeconds  uint
	FertilizerValvuleRelayPin string
	WaitDuration              time.Duration
	ClarifyDuration           time.Duration
}

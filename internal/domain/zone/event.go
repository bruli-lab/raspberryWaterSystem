package zone

import "github.com/bruli/raspberryWaterSystem/internal/cqs"

const (
	ExecutedEventName               = "v1.zone.executed"
	IgnoredEventName                = "v1.zone.ignored"
	FertilizerZoneExecutedEventName = "v1.fertilizer.zone.executed"
)

type Executed struct {
	cqs.BasicEvent
	ZoneID               string
	ZoneName             string
	Seconds              uint
	StabilizationSeconds uint
	RelayPins            []string
}

type Ignored struct {
	cqs.BasicEvent
	ZoneName string
	Reason   string
}

type FertilizerZoneExecuted struct {
	cqs.BasicEvent
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

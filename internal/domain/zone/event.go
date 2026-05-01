package zone

import "github.com/bruli/raspberryWaterSystem/internal/cqs"

const (
	ExecutedEventName = "v1.zone.executed"
	IgnoredEventName  = "v1.zone.ignored"
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

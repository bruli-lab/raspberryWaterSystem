package gpio

import (
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/host/v3/rpi"
)

var relays = map[string]gpio.PinIO{
	"master":            rpi.P1_12,
	"fertilizerPump":    rpi.P1_11,
	"big":               rpi.P1_16,
	"small":             rpi.P1_18,
	"air":               rpi.P1_10,
	"clean":             rpi.P1_8,
	"22":                rpi.P1_15,
	"fertilizerValvule": rpi.P1_13,
}

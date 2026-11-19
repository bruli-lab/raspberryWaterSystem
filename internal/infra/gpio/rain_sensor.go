package gpio

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/host/v3/sysfs"
)

const (
	RainReference = 600

	rainSensorSPIBus        = 0
	rainSensorSPIChipSelect = 0
	rainSensorADCChannel    = 0
	rainSensorSPIFrequency  = physic.MegaHertz
	rainSensorWordBits      = 8
	adcStartBit             = 0x01
	adcSingleEndedMode      = 0x08
	adcHighBitsMask         = 0x03
)

type SPIPortOpener func() (spi.PortCloser, error)

type RainSensor struct {
	m      sync.Mutex
	open   SPIPortOpener
	tracer trace.Tracer
}

func (r *RainSensor) Find(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	_, span := r.tracer.Start(ctx, "RainSensor.Find")
	defer span.End()
	r.m.Lock()
	defer r.m.Unlock()
	value, err := r.readADC()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	span.SetStatus(codes.Ok, "rain sensor found")
	return value > RainReference, nil
}

func (r *RainSensor) readADC() (int, error) {
	port, err := r.open()
	if err != nil {
		return 0, RainSensorError{operation: "open spi port", err: err}
	}
	defer func() {
		_ = port.Close()
	}()
	conn, err := port.Connect(rainSensorSPIFrequency, spi.Mode0, rainSensorWordBits)
	if err != nil {
		return 0, RainSensorError{operation: "connect to spi port", err: err}
	}
	request := []byte{adcStartBit, (adcSingleEndedMode + rainSensorADCChannel) << 4, 0}
	response := make([]byte, len(request))
	if err = conn.Tx(request, response); err != nil {
		return 0, RainSensorError{operation: "read adc", err: err}
	}
	return int(response[1]&adcHighBitsMask)<<8 | int(response[2]), nil
}

func NewRainSensor(tracer trace.Tracer) *RainSensor {
	return NewRainSensorWithPort(func() (spi.PortCloser, error) {
		return sysfs.NewSPI(rainSensorSPIBus, rainSensorSPIChipSelect)
	}, tracer)
}

func NewRainSensorWithPort(open SPIPortOpener, tracer trace.Tracer) *RainSensor {
	return &RainSensor{open: open, tracer: tracer}
}

type RainSensorError struct {
	operation string
	err       error
}

func (r RainSensorError) Error() string {
	return fmt.Sprintf("rain sensor failed to %s: %s", r.operation, r.err)
}

func (r RainSensorError) Unwrap() error {
	return r.err
}

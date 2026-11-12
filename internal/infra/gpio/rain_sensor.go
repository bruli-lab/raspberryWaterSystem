package gpio

import (
	"context"
	"sync"

	"github.com/stianeikeland/go-rpio/v4"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const RainReference = 600

type RainSensor struct {
	m      sync.Mutex
	tracer trace.Tracer
}

func (r *RainSensor) Find(ctx context.Context) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
		_, span := r.tracer.Start(ctx, "RainSensor.Find")
		defer span.End()
		r.m.Lock()
		defer r.m.Unlock()
		if err := rpio.Open(); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}

		defer func() {
			_ = rpio.Close()
		}()

		if err := rpio.SpiBegin(rpio.Spi0); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}

		rpio.SpiSpeed(1000000)
		rpio.SpiChipSelect(0)
		channel := byte(0)
		data := []byte{1, (8 + channel) << 4, 0}

		rpio.SpiExchange(data)

		value := int(data[1]&3)<<8 + int(data[2])
		defer rpio.SpiEnd(rpio.Spi0)

		span.SetStatus(codes.Ok, "rain sensor found")
		return value > RainReference, nil
	}
}

func NewRainSensor(tracer trace.Tracer) *RainSensor {
	return &RainSensor{tracer: tracer}
}

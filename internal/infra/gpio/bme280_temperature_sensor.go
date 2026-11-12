package gpio

import (
	"context"
	"fmt"
	"sync"

	"github.com/bruli/raspberryWaterSystem/internal/domain/weather"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/devices/v3/bmxx80"
	"periph.io/x/host/v3"
)

type Bme280TemperatureSensor struct {
	bus    i2c.BusCloser
	m      sync.Mutex
	tracer trace.Tracer
}

func (b *Bme280TemperatureSensor) Find(ctx context.Context) (weather.Temperature, weather.Humidity, error) {
	select {
	case <-ctx.Done():
		return 0, 0, ctx.Err()
	default:
		_, span := b.tracer.Start(ctx, "Bme280TemperatureSensor.Find")
		defer span.End()
		b.m.Lock()
		defer b.m.Unlock()
		sensor, err := bmxx80.NewI2C(b.bus, 0x76, &bmxx80.DefaultOpts)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return 0, 0, fmt.Errorf("failed to create temperature sensor: %w", err)
		}
		defer func() {
			_ = sensor.Halt()
		}()
		var env physic.Env
		if err = sensor.Sense(&env); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return 0, 0, fmt.Errorf("failed to sense temperature: %w", err)
		}

		temp := weather.Temperature(env.Temperature.Celsius())
		hum := weather.Humidity(env.Humidity) / 100000
		span.SetStatus(codes.Ok, "temperature found")
		return temp, hum, nil
	}
}

func NewBme280TemperatureSensor(tracer trace.Tracer) (*Bme280TemperatureSensor, error) {
	if _, err := host.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize temperature sensor host: %v", err)
	}
	bus, err := i2creg.Open("")
	if err != nil {
		return nil, fmt.Errorf("failed to open i2c bus: %v", err)
	}
	return &Bme280TemperatureSensor{bus: bus, tracer: tracer}, nil
}

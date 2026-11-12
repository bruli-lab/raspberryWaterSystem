package memory

import (
	"context"
	"sync"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/weather"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type lastKnown[T any] struct {
	m      sync.Mutex
	value  T
	readAt time.Time
	found  bool
	maxAge time.Duration
	now    func() time.Time
}

func (l *lastKnown[T]) remember(value T) {
	l.m.Lock()
	defer l.m.Unlock()
	l.value = value
	l.readAt = l.now()
	l.found = true
}

func (l *lastKnown[T]) recall() (T, bool) {
	l.m.Lock()
	defer l.m.Unlock()
	if !l.found || l.now().Sub(l.readAt) > l.maxAge {
		var zero T
		return zero, false
	}
	return l.value, true
}

type temperatureReading struct {
	temperature weather.Temperature
	humidity    weather.Humidity
}

type LastKnownTemperatureRepository struct {
	sensor app.TemperatureRepository
	last   *lastKnown[temperatureReading]
	tracer trace.Tracer
}

func (l LastKnownTemperatureRepository) Find(ctx context.Context) (weather.Temperature, weather.Humidity, error) {
	ctx, span := l.tracer.Start(ctx, "LastKnownTemperatureRepository.Find")
	defer span.End()
	temp, hum, err := l.sensor.Find(ctx)
	if err == nil {
		l.last.remember(temperatureReading{temperature: temp, humidity: hum})
		span.SetStatus(codes.Ok, "temperature read")
		return temp, hum, nil
	}
	span.RecordError(err)
	reading, ok := l.last.recall()
	if !ok {
		span.SetStatus(codes.Error, err.Error())
		return 0, 0, err
	}
	span.SetStatus(codes.Ok, "last known temperature used")
	return reading.temperature, reading.humidity, nil
}

func NewLastKnownTemperatureRepository(sensor app.TemperatureRepository, maxAge time.Duration, now func() time.Time, tracer trace.Tracer) LastKnownTemperatureRepository {
	return LastKnownTemperatureRepository{
		sensor: sensor,
		last:   &lastKnown[temperatureReading]{maxAge: maxAge, now: now},
		tracer: tracer,
	}
}

type LastKnownRainRepository struct {
	sensor app.RainRepository
	last   *lastKnown[bool]
	tracer trace.Tracer
}

func (l LastKnownRainRepository) Find(ctx context.Context) (bool, error) {
	ctx, span := l.tracer.Start(ctx, "LastKnownRainRepository.Find")
	defer span.End()
	raining, err := l.sensor.Find(ctx)
	if err == nil {
		l.last.remember(raining)
		span.SetStatus(codes.Ok, "rain read")
		return raining, nil
	}
	span.RecordError(err)
	raining, ok := l.last.recall()
	if !ok {
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	span.SetStatus(codes.Ok, "last known rain used")
	return raining, nil
}

func NewLastKnownRainRepository(sensor app.RainRepository, maxAge time.Duration, now func() time.Time, tracer trace.Tracer) LastKnownRainRepository {
	return LastKnownRainRepository{
		sensor: sensor,
		last:   &lastKnown[bool]{maxAge: maxAge, now: now},
		tracer: tracer,
	}
}

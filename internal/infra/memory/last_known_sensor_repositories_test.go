package memory_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/weather"
	"github.com/bruli/raspberryWaterSystem/internal/infra/memory"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

type sensorReading struct {
	temperature weather.Temperature
	humidity    weather.Humidity
	raining     bool
	err         error
}

type scriptedSensor struct {
	m        sync.Mutex
	readings []sensorReading
}

func (s *scriptedSensor) next() sensorReading {
	s.m.Lock()
	defer s.m.Unlock()
	r := s.readings[0]
	s.readings = s.readings[1:]
	return r
}

type scriptedTemperatureSensor struct{ *scriptedSensor }

func (s scriptedTemperatureSensor) Find(_ context.Context) (weather.Temperature, weather.Humidity, error) {
	r := s.next()
	return r.temperature, r.humidity, r.err
}

type scriptedRainSensor struct{ *scriptedSensor }

func (s scriptedRainSensor) Find(_ context.Context) (bool, error) {
	r := s.next()
	return r.raining, r.err
}

type fakeClock struct {
	m   sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.m.Lock()
	defer c.m.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.m.Lock()
	defer c.m.Unlock()
	c.now = c.now.Add(d)
}

func TestLastKnownSensorRepositories(t *testing.T) {
	errSensor := errors.New("i2c read failed")
	tests := []struct {
		name                string
		readings            []sensorReading
		elapsed             time.Duration
		expectedTemperature weather.Temperature
		expectedHumidity    weather.Humidity
		expectedRaining     bool
		expectedErr         bool
	}{
		{
			name:                "and the sensor fails after a valid reading, then it returns the last valid reading",
			readings:            []sensorReading{{temperature: 22.5, humidity: 60, raining: true}, {err: errSensor}},
			elapsed:             10 * time.Minute,
			expectedTemperature: 22.5,
			expectedHumidity:    60,
			expectedRaining:     true,
		},
		{
			name:        "and the sensor fails after a reading older than the max age, then it returns the sensor error",
			readings:    []sensorReading{{temperature: 22.5, humidity: 60, raining: true}, {err: errSensor}},
			elapsed:     2 * time.Hour,
			expectedErr: true,
		},
		{
			name:        "and the sensor fails without any previous reading, then it returns the sensor error",
			readings:    []sensorReading{{err: errSensor}, {err: errSensor}},
			expectedErr: true,
		},
		{
			name:                "and the sensor reads again, then it returns the new reading",
			readings:            []sensorReading{{temperature: 22.5, humidity: 60, raining: true}, {temperature: 18, humidity: 70}},
			expectedTemperature: 18,
			expectedHumidity:    70,
		},
	}
	for _, tt := range tests {
		t.Run(`Given last known temperature and rain repositories,
		when Find method is called twice `+tt.name, func(t *testing.T) {
			t.Parallel()
			clock := &fakeClock{now: time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)}
			tracer := noop.NewTracerProvider().Tracer("test")
			temperatureReadings := append([]sensorReading(nil), tt.readings...)
			rainReadings := append([]sensorReading(nil), tt.readings...)
			tempRepo := memory.NewLastKnownTemperatureRepository(scriptedTemperatureSensor{&scriptedSensor{readings: temperatureReadings}}, time.Hour, clock.Now, tracer)
			rainRepo := memory.NewLastKnownRainRepository(scriptedRainSensor{&scriptedSensor{readings: rainReadings}}, time.Hour, clock.Now, tracer)
			_, _, _ = tempRepo.Find(t.Context())
			_, _ = rainRepo.Find(t.Context())
			clock.advance(tt.elapsed)

			temp, hum, tempErr := tempRepo.Find(t.Context())
			raining, rainErr := rainRepo.Find(t.Context())

			if tt.expectedErr {
				require.ErrorIs(t, tempErr, errSensor)
				require.ErrorIs(t, rainErr, errSensor)
				return
			}
			require.NoError(t, tempErr)
			require.NoError(t, rainErr)
			require.Equal(t, tt.expectedTemperature, temp)
			require.Equal(t, tt.expectedHumidity, hum)
			require.Equal(t, tt.expectedRaining, raining)
		})
	}
}

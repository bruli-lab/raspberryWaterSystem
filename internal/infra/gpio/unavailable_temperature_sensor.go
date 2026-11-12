package gpio

import (
	"context"
	"fmt"

	"github.com/bruli/raspberryWaterSystem/internal/domain/weather"
)

type UnavailableTemperatureSensor struct {
	cause error
}

func (u UnavailableTemperatureSensor) Find(_ context.Context) (weather.Temperature, weather.Humidity, error) {
	return 0, 0, fmt.Errorf("temperature sensor unavailable: %w", u.cause)
}

func NewUnavailableTemperatureSensor(cause error) UnavailableTemperatureSensor {
	return UnavailableTemperatureSensor{cause: cause}
}

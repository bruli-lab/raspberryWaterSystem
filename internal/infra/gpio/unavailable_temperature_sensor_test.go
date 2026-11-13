package gpio_test

import (
	"errors"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/infra/gpio"
	"github.com/stretchr/testify/require"
)

func TestUnavailableTemperatureSensorFind(t *testing.T) {
	t.Run(`Given an UnavailableTemperatureSensor built with an initialization error,
	when Find method is called,
	then it returns an error wrapping the initialization error instead of panicking`, func(t *testing.T) {
		t.Parallel()
		cause := errors.New("i2c bus not found")
		sensor := gpio.NewUnavailableTemperatureSensor(cause)

		temp, hum, err := sensor.Find(t.Context())

		require.ErrorIs(t, err, cause)
		require.Zero(t, temp)
		require.Zero(t, hum)
	})
}

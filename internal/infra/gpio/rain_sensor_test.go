package gpio_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/infra/gpio"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"periph.io/x/conn/v3/conntest"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spitest"
)

func adcReading(value int) []byte {
	return []byte{0, byte(value >> 8), byte(value)}
}

func TestRainSensorFind(t *testing.T) {
	errOpen := errors.New("open /dev/spidev0.0: permission denied")
	channelZeroRequest := []byte{0x01, 0x80, 0x00}
	tests := []struct {
		name            string
		openErr         error
		response        []byte
		cancelled       bool
		expectedRaining bool
		expectedErr     error
	}{
		{
			name:        "and the context is cancelled, then it returns the context error without opening the port",
			cancelled:   true,
			expectedErr: context.Canceled,
		},
		{
			name:        "and the spi port cannot be opened, then it returns a rain sensor error",
			openErr:     errOpen,
			expectedErr: gpio.RainSensorError{},
		},
		{
			name:     "and the adc reads a dry value, then it is not raining",
			response: adcReading(120),
		},
		{
			name:     "and the adc reads the reference value, then it is not raining",
			response: adcReading(gpio.RainReference),
		},
		{
			name:            "and the adc reads one above the reference value, then it is raining",
			response:        adcReading(gpio.RainReference + 1),
			expectedRaining: true,
		},
		{
			name:            "and the adc reads the maximum value, then it is raining",
			response:        adcReading(1023),
			expectedRaining: true,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a RainSensor reading channel 0 of the adc through spi,
		when Find method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tt.cancelled {
				cancel()
			}
			opened := 0
			port := &spitest.Playback{Playback: conntest.Playback{
				Ops:       []conntest.IO{{W: channelZeroRequest, R: tt.response}},
				DontPanic: true,
			}}
			sensor := gpio.NewRainSensorWithPort(func() (spi.PortCloser, error) {
				opened++
				if tt.openErr != nil {
					return nil, tt.openErr
				}
				return port, nil
			}, noop.NewTracerProvider().Tracer(""))

			raining, err := sensor.Find(ctx)

			if tt.cancelled {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Zero(t, opened)
				return
			}
			if tt.expectedErr != nil {
				require.IsType(t, tt.expectedErr, err)
				require.ErrorIs(t, err, tt.openErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expectedRaining, raining)
			require.Equal(t, 1, port.Count)
		})
	}
}

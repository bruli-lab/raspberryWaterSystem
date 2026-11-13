//go:build prod

package main

import (
	"log/slog"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/infra/gpio"
	"go.opentelemetry.io/otel/trace"
)

func temperatureRepository(tracer trace.Tracer, log *slog.Logger) app.TemperatureRepository {
	repo, err := gpio.NewBme280TemperatureSensor(tracer)
	if err != nil {
		log.Error("failed initializing temperature sensor", slog.String("error", err.Error()))
		return gpio.NewUnavailableTemperatureSensor(err)
	}
	return repo
}

func pinsExecutor(tracer trace.Tracer) relaysExecutor {
	return gpio.NewPinsExecutor(tracer)
}

func rainRepository(tracer trace.Tracer) app.RainRepository {
	return gpio.NewRainSensor(tracer)
}

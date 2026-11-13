package solar

//go:generate go tool moq -out light_repository_mock_test.go -pkg solar_test ../../app LightRepository

import (
	"context"
	"log/slog"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type FallbackLightRepository struct {
	primary, fallback app.LightRepository
	tracer            trace.Tracer
	log               *slog.Logger
}

func (f FallbackLightRepository) Find(ctx context.Context, date time.Time) (*status.Light, error) {
	ctx, span := f.tracer.Start(ctx, "FallbackLightRepository.Find")
	defer span.End()
	light, err := f.primary.Find(ctx, date)
	if err == nil {
		span.SetStatus(codes.Ok, "light found")
		return light, nil
	}
	span.RecordError(err)
	f.log.WarnContext(ctx, "failed finding light, using fallback", slog.String("error", err.Error()))
	light, err = f.fallback.Find(ctx, date)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "light found with fallback")
	return light, nil
}

func NewFallbackLightRepository(primary, fallback app.LightRepository, tracer trace.Tracer, log *slog.Logger) FallbackLightRepository {
	return FallbackLightRepository{primary: primary, fallback: fallback, tracer: tracer, log: log}
}

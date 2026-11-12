package app

import (
	"context"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/weather"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const UpdateStatusCmdName = "updateStatus"

type UpdateStatusCmd struct {
	Weather weather.Weather
}

func (u UpdateStatusCmd) Name() string {
	return UpdateStatusCmdName
}

type UpdateStatus struct {
	sr     StatusRepository
	lr     LightRepository
	tracer trace.Tracer
}

func (u UpdateStatus) Handle(ctx context.Context, cmd cqs.Command) ([]cqs.Event, error) {
	ctx, span := u.tracer.Start(ctx, "UpdateStatusCmd")
	defer span.End()
	co, ok := cmd.(UpdateStatusCmd)
	if !ok {
		err := cqs.NewInvalidCommandError(UpdateStatusCmdName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	li, err := u.lr.Find(ctx, time.Now())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	current, err := u.sr.Find(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	current.Update(co.Weather, li)
	if err = u.sr.Update(ctx, current); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "status updated")
	return nil, nil
}

func NewUpdateStatus(sr StatusRepository, lr LightRepository, tracer trace.Tracer) UpdateStatus {
	return UpdateStatus{sr: sr, lr: lr, tracer: tracer}
}

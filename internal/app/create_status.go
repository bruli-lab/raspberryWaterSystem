package app

import (
	"context"
	"errors"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"github.com/bruli/raspberryWaterSystem/internal/domain/weather"
)

var ErrStatusAlreadyExist = errors.New("status already exist")

const CreateStatusCmdName = "createStatus"

type CreateStatusCmd struct {
	StartedAt time.Time
	Weather   weather.Weather
}

func (c CreateStatusCmd) Name() string {
	return CreateStatusCmdName
}

type CreateStatus struct {
	sr     StatusRepository
	lr     LightRepository
	tracer trace.Tracer
}

func (c CreateStatus) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := c.tracer.Start(ctx, "CreateStatusCmd")
	defer span.End()
	co, ok := cmd.(CreateStatusCmd)
	if !ok {
		err := cqs.NewInvalidCommandError(CreateStatusCmdName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	_, err := c.sr.Find(ctx)
	if err == nil {
		span.RecordError(ErrStatusAlreadyExist)
		span.SetStatus(codes.Error, ErrStatusAlreadyExist.Error())
		return nil, ErrStatusAlreadyExist
	}
	switch {
	case errors.As(err, &errs.NotFoundError{}):
		li, err := c.lr.Find(ctx, time.Now().UTC())
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		st := status.New(co.StartedAt, co.Weather, li)
		if err = c.sr.Save(ctx, *st); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.SetStatus(codes.Ok, "status created")
		return nil, nil
	default:
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
}

func NewCreateStatus(sr StatusRepository, lr LightRepository, tracer trace.Tracer) CreateStatus {
	return CreateStatus{sr: sr, lr: lr, tracer: tracer}
}

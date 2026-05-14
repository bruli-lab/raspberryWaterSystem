package app

import (
	"context"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const ChangeCleanStatusCmdName = "ChangeCleanStatusCmd"

type ChangeCleanStatusCmd struct {
	Clean bool
}

func (c ChangeCleanStatusCmd) Name() string {
	return ChangeCleanStatusCmdName
}

type ChangeCleanStatus struct {
	repo   StatusRepository
	tracer trace.Tracer
}

func (c ChangeCleanStatus) Handle(ctx context.Context, cmd cqs.Command) ([]cqs.Event, error) {
	ctx, span := c.tracer.Start(ctx, "ChangeCleanStatusCmd")
	defer span.End()
	co, ok := cmd.(ChangeCleanStatusCmd)
	if !ok {
		err := cqs.NewInvalidCommandError(ChangeCleanStatusCmdName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	st, err := c.repo.Find(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	switch {
	case co.Clean:
		st.Cleaning()
	default:
		st.DoneCleaning()
	}
	if err := c.repo.Update(ctx, st); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "clean status changed")
	return nil, nil
}

func NewChangeCleanStatus(repo StatusRepository, tracer trace.Tracer) *ChangeCleanStatus {
	return &ChangeCleanStatus{repo: repo, tracer: tracer}
}

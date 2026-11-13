package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const FindStatusQueryName = "findStatus"

type FindStatusQuery struct{}

func (f FindStatusQuery) Name() string {
	return FindStatusQueryName
}

type FindStatus struct {
	sr     StatusRepository
	tracer trace.Tracer
}

func (f FindStatus) Handle(ctx context.Context, _ cqs.Query) (any, error) {
	ctx, span := f.tracer.Start(ctx, "FindStatusQuery")
	defer span.End()
	find, err := f.sr.Find(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "status found")
	return find, nil
}

func NewFindStatus(sr StatusRepository, tracer trace.Tracer) FindStatus {
	return FindStatus{sr: sr, tracer: tracer}
}

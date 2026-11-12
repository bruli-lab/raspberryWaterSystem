package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const FindZonesQueryName = "findZones"

type FindZonesQuery struct{}

func (f FindZonesQuery) Name() string {
	return FindZonesQueryName
}

type FindZones struct {
	zr     ZoneRepository
	tracer trace.Tracer
}

func (f FindZones) Handle(ctx context.Context, _ cqs.Query) (any, error) {
	ctx, span := f.tracer.Start(ctx, "FindZones")
	defer span.End()
	all, err := f.zr.FindAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "zones found")
	return all, nil
}

func NewFindZones(zr ZoneRepository, tracer trace.Tracer) *FindZones {
	return &FindZones{zr: zr, tracer: tracer}
}

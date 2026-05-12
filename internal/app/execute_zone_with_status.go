package app

import (
	"context"
	"fmt"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const ExecuteZoneWithStatusCmdName = "executeZoneWithStatus"

type ExecuteZoneWithStatusCmd struct {
	Seconds        uint
	ZoneID         string
	WithFertilizer bool
}

func (e ExecuteZoneWithStatusCmd) Name() string {
	return ExecuteZoneWithStatusCmdName
}

type ExecuteZoneWithStatus struct {
	zr     ZoneRepository
	st     StatusRepository
	tracer trace.Tracer
}

func (e ExecuteZoneWithStatus) Handle(ctx context.Context, cmd cqs.Command) ([]cqs.Event, error) {
	ctx, span := e.tracer.Start(ctx, "ExecuteZoneWithStatusCmd")
	defer span.End()
	co, ok := cmd.(ExecuteZoneWithStatusCmd)
	if !ok {
		err := cqs.NewInvalidCommandError(ExecuteZoneWithStatusCmdName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	zo, err := e.zr.FindByID(ctx, co.ZoneID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	st, err := e.st.Find(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	switch {
	case co.WithFertilizer:
		fzo := zone.NewFertilizerZone(zo)
		if err = fzo.ExecuteWithStatus(st.IsActive(), st.Weather().IsRaining(), co.Seconds); err != nil {
			err = ExecuteZoneWithStatusError{m: err.Error()}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.SetStatus(codes.Ok, "fertilizer zone executed")
		return fzo.Events(), nil
	default:
	}

	if err = zo.ExecuteWithStatus(st.IsActive(), st.Weather().IsRaining(), co.Seconds); err != nil {
		err = ExecuteZoneWithStatusError{m: err.Error()}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "zone executed")
	return zo.Events(), nil
}

func NewExecuteZoneWithStatus(zr ZoneRepository, st StatusRepository, tracer trace.Tracer) *ExecuteZoneWithStatus {
	return &ExecuteZoneWithStatus{zr: zr, st: st, tracer: tracer}
}

type ExecuteZoneWithStatusError struct {
	m string
}

func (e ExecuteZoneWithStatusError) Error() string {
	return fmt.Sprintf("failed to execute zone with status: %s", e.m)
}

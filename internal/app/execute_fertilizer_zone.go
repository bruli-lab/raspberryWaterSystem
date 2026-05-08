package app

import (
	"context"
	"fmt"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const ExecuteFertilizerZoneCmdName = "executeFertilizerZone"

type ExecuteFertilizerZoneCmd struct {
	Seconds uint
	ZoneID  string
}

func (e ExecuteFertilizerZoneCmd) Name() string {
	return ExecuteFertilizerZoneCmdName
}

type ExecuteFertilizerZone struct {
	zr     ZoneRepository
	tracer trace.Tracer
}

func (e ExecuteFertilizerZone) Handle(ctx context.Context, cmd cqs.Command) ([]cqs.Event, error) {
	ctx, span := e.tracer.Start(ctx, "ExecuteFertilizerZoneCmd")
	defer span.End()
	co, ok := cmd.(ExecuteFertilizerZoneCmd)
	if !ok {
		err := cqs.NewInvalidCommandError(ExecuteFertilizerZoneCmdName, cmd.Name())
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
	fzo := zone.NewFertilizerZone(zo)

	if err = fzo.Execute(co.Seconds); err != nil {
		err = ExecuteFertilizerZoneError{m: err.Error()}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "fertilizer zone executed")
	return fzo.Events(), nil
}

func NewExecuteFertilizerZone(zr ZoneRepository, tracer trace.Tracer) ExecuteFertilizerZone {
	return ExecuteFertilizerZone{zr: zr, tracer: tracer}
}

type ExecuteFertilizerZoneError struct {
	m string
}

func (e ExecuteFertilizerZoneError) Error() string {
	return fmt.Sprintf("failed to execute zone: %s", e.m)
}

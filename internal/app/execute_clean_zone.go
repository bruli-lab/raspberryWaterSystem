package app

import (
	"context"
	"fmt"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const ExecuteCleanZoneCmdName = "executeCleanZone"

type ExecuteCleanZoneCmd struct {
	ZoneID string
}

func (e ExecuteCleanZoneCmd) Name() string {
	return ExecuteCleanZoneCmdName
}

type ExecuteCleanZone struct {
	zr     ZoneRepository
	tracer trace.Tracer
}

func (e ExecuteCleanZone) Handle(ctx context.Context, cmd cqs.Command) ([]cqs.Event, error) {
	ctx, span := e.tracer.Start(ctx, "ExecuteCleanZoneCmd")
	defer span.End()
	co, ok := cmd.(ExecuteCleanZoneCmd)
	if !ok {
		err := cqs.NewInvalidCommandError(ExecuteCleanZoneCmdName, cmd.Name())
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
	czo := zone.NewCleanZone(zo)
	czo.Execute()
	span.SetStatus(codes.Ok, "clean zone executed")
	return czo.Events(), nil
}

func NewExecuteCleanZone(zr ZoneRepository, tracer trace.Tracer) ExecuteCleanZone {
	return ExecuteCleanZone{zr: zr, tracer: tracer}
}

type ExecuteCleanZoneError struct {
	m string
}

func (e ExecuteCleanZoneError) Error() string {
	return fmt.Sprintf("failed to execute clean zone: %s", e.m)
}

package app

import (
	"context"
	"fmt"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const ExecuteZoneCmdName = "executeZone"

type ExecuteZoneCmd struct {
	Seconds uint
	ZoneID  string
}

func (e ExecuteZoneCmd) Name() string {
	return ExecuteZoneCmdName
}

type ExecuteZone struct {
	zr     ZoneRepository
	stRepo StatusRepository
	tracer trace.Tracer
}

func (e ExecuteZone) Handle(ctx context.Context, cmd cqs.Command) ([]cqs.Event, error) {
	ctx, span := e.tracer.Start(ctx, "ExecuteZoneCmd")
	defer span.End()
	co, ok := cmd.(ExecuteZoneCmd)
	if !ok {
		err := cqs.NewInvalidCommandError(ExecuteZoneCmdName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	st, err := e.stRepo.Find(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if st.IsClean() {
		err := ExecuteZoneError{m: "system on clean mode"}
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
	if err = zo.Execute(co.Seconds); err != nil {
		err = ExecuteZoneError{m: err.Error()}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "zone executed")
	return zo.Events(), nil
}

func NewExecuteZone(zr ZoneRepository, statusRepo StatusRepository, tracer trace.Tracer) ExecuteZone {
	return ExecuteZone{zr: zr, stRepo: statusRepo, tracer: tracer}
}

type ExecuteZoneError struct {
	m string
}

func (e ExecuteZoneError) Error() string {
	return fmt.Sprintf("failed to execute zone: %s", e.m)
}

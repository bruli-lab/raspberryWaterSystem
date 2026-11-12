package app

import (
	"context"
	"fmt"
	"sync"

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
	sr     StatusRepository
	m      *sync.Mutex
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
	if err = e.startCleaning(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	czo := zone.NewCleanZone(zo)
	czo.Execute()
	span.SetStatus(codes.Ok, "clean zone executed")
	return czo.Events(), nil
}

func (e ExecuteCleanZone) startCleaning(ctx context.Context) error {
	e.m.Lock()
	defer e.m.Unlock()
	st, err := e.sr.Find(ctx)
	if err != nil {
		return err
	}
	if st.IsClean() {
		return CleanInProgressError{}
	}
	st.Cleaning()
	return e.sr.Update(ctx, st)
}

func NewExecuteCleanZone(zr ZoneRepository, sr StatusRepository, tracer trace.Tracer) ExecuteCleanZone {
	return ExecuteCleanZone{zr: zr, sr: sr, m: &sync.Mutex{}, tracer: tracer}
}

type CleanInProgressError struct{}

func (CleanInProgressError) Error() string {
	return "a clean cycle is already in progress"
}

type ExecuteCleanZoneError struct {
	m string
}

func (e ExecuteCleanZoneError) Error() string {
	return fmt.Sprintf("failed to execute clean zone: %s", e.m)
}

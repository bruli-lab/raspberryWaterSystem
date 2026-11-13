package app

import (
	"context"
	"sync"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
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

func (e ExecuteCleanZone) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
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

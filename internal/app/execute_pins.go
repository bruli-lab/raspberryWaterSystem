package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const ExecutePinsCmdName = "executePins"

type ExecutePinsCmd struct {
	Seconds    uint
	Pins       []string
	CleanCycle bool
}

func (e ExecutePinsCmd) Name() string {
	return ExecutePinsCmdName
}

type ExecutePins struct {
	pe     PinExecutor
	sr     StatusRepository
	tracer trace.Tracer
}

func (e ExecutePins) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := e.tracer.Start(ctx, "ExecutePinsCmd")
	defer span.End()
	co, ok := cmd.(ExecutePinsCmd)
	if !ok {
		err := cqs.NewInvalidCommandError(ExecutePinsCmdName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if !co.CleanCycle {
		if err := e.ensureNotCleaning(ctx); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}
	if err := e.pe.Execute(ctx, co.Seconds, co.Pins); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return nil, nil
}

func (e ExecutePins) ensureNotCleaning(ctx context.Context) error {
	st, err := e.sr.Find(ctx)
	if err != nil {
		return err
	}
	if st.IsClean() {
		return CleanInProgressError{}
	}
	return nil
}

func NewExecutePins(pe PinExecutor, sr StatusRepository, tracer trace.Tracer) ExecutePins {
	return ExecutePins{pe: pe, sr: sr, tracer: tracer}
}

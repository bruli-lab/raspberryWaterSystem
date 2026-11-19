package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	TestRelayCmdName = "testRelay"
	RelayTestSeconds = 5
)

type TestRelayCmd struct {
	RelayID int
}

func (t TestRelayCmd) Name() string {
	return TestRelayCmdName
}

type TestRelay struct {
	pe     PinExecutor
	sr     StatusRepository
	tracer trace.Tracer
}

func (t TestRelay) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := t.tracer.Start(ctx, "TestRelayCmd")
	defer span.End()
	co, ok := cmd.(TestRelayCmd)
	if !ok {
		err := cqs.NewInvalidCommandError(TestRelayCmdName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	relay, err := zone.ParseRelay(co.RelayID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	st, err := t.sr.Find(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if st.IsClean() {
		err = CleanInProgressError{}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err = t.pe.Execute(ctx, RelayTestSeconds, []string{relay.Pin()}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "relay tested")
	return nil, nil
}

func NewTestRelay(pe PinExecutor, sr StatusRepository, tracer trace.Tracer) TestRelay {
	return TestRelay{pe: pe, sr: sr, tracer: tracer}
}

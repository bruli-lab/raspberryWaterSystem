package worker

import (
	"context"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type EventsWriter interface {
	Save(ctx context.Context, ev *outbox.Event) error
}

func SaveTerraceWeather(ctx context.Context, qh cqs.QueryHandler, events EventsWriter, now time.Time, tracer trace.Tracer) error {
	ctx, span := tracer.Start(ctx, "SaveTerraceWeather worker")
	defer span.End()
	result, err := qh.Handle(ctx, app.FindStatusQuery{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	st, ok := result.(status.Status)
	if !ok {
		err = UnexpectedQueryResultError{result: result}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	ev, err := outbox.NewFromWeather(ctx, outbox.Weather{
		Temperature: st.Weather().Temperature().Float32(),
		IsRaining:   st.Weather().IsRaining(),
		Humidity:    st.Weather().Humidity().Float32(),
	}, now)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err = events.Save(ctx, ev); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "terrace weather saved")
	return nil
}

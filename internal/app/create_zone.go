package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/davecgh/go-spew/spew"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const CreateZoneCmdName = "createZone"

type CreateZoneCmd struct {
	ID, ZoneName string
	Relays       []int
}

func (c CreateZoneCmd) Name() string {
	return CreateZoneCmdName
}

type CreateZone struct {
	zr     ZoneRepository
	tracer trace.Tracer
}

func (c CreateZone) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := c.tracer.Start(ctx, "CreateZoneCmd")
	defer span.End()
	co, ok := cmd.(CreateZoneCmd)
	if !ok {
		err := cqs.NewInvalidCommandError(CreateZoneCmdName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	_, err := c.zr.FindByID(ctx, co.ID)
	if err == nil {
		err = CreateZoneError{msg: fmt.Sprintf("a zone with id %s, already exists", co.ID)}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	switch {
	case errors.As(err, &errs.NotFoundError{}):
		if err := c.createZone(ctx, co); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span.SetStatus(codes.Ok, "zone created")
		return nil, nil
	default:
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
}

func (c CreateZone) createZone(ctx context.Context, co CreateZoneCmd) error {
	relays := make([]zone.Relay, len(co.Relays))
	for i, re := range co.Relays {
		r, err := zone.ParseRelay(re)
		if err != nil {
			return CreateZoneError{msg: err.Error()}
		}
		relays[i] = r
	}
	zo, err := zone.New(co.ID, co.ZoneName, relays)
	if err != nil {
		return CreateZoneError{msg: err.Error()}
	}
	return c.zr.Save(ctx, zo)
}

func NewCreateZone(zr ZoneRepository, tracer trace.Tracer) CreateZone {
	return CreateZone{zr: zr, tracer: tracer}
}

type CreateZoneError struct {
	msg string
}

func (c CreateZoneError) Error() string {
	return spew.Sprintf("failed to create zone: %s", c.msg)
}

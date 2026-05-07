package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/davecgh/go-spew/spew"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const CreateFertilizerZoneCmdName = "createFertilizerZone"

type CreateFertilizerZoneCmd struct {
	ID, ZoneName string
	Relays       []int
}

func (c CreateFertilizerZoneCmd) Name() string {
	return CreateFertilizerZoneCmdName
}

type CreateFertilizerZone struct {
	zr     FertilizerZoneRepository
	tracer trace.Tracer
}

func (c CreateFertilizerZone) Handle(ctx context.Context, cmd cqs.Command) ([]cqs.Event, error) {
	ctx, span := c.tracer.Start(ctx, "CreateFertilizerZoneCmd")
	defer span.End()
	co, ok := cmd.(CreateFertilizerZoneCmd)
	if !ok {
		err := cqs.NewInvalidCommandError(CreateFertilizerZoneCmdName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	_, err := c.zr.FindByID(ctx, co.ID)
	if err == nil {
		err = CreateFertilizerZoneError{msg: fmt.Sprintf("a fertilizer zone with id %s, already exists", co.ID)}
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
		span.SetStatus(codes.Ok, "fertilizer zone created")
		return nil, nil
	default:
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
}

func (c CreateFertilizerZone) createZone(ctx context.Context, co CreateFertilizerZoneCmd) error {
	relays := make([]zone.Relay, len(co.Relays))
	for i, re := range co.Relays {
		r, err := zone.ParseRelay(re)
		if err != nil {
			return CreateFertilizerZoneError{msg: err.Error()}
		}
		relays[i] = r
	}
	zo, err := zone.New(co.ID, co.ZoneName, relays)
	if err != nil {
		return CreateFertilizerZoneError{msg: err.Error()}
	}
	return c.zr.Save(ctx, zone.NewFertilizerZone(zo))
}

func NewCreateFertilizerZone(zr FertilizerZoneRepository, tracer trace.Tracer) *CreateFertilizerZone {
	return &CreateFertilizerZone{zr: zr, tracer: tracer}
}

type CreateFertilizerZoneError struct {
	msg string
}

func (c CreateFertilizerZoneError) Error() string {
	return spew.Sprintf("failed to create fertilizer zone: %s", c.msg)
}

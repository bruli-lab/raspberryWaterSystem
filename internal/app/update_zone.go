package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const UpdateZoneCommandName = "updateZone"

type UpdateZoneCommand struct {
	ID, ZoneName string
	Relays       []int
}

func (u UpdateZoneCommand) Name() string {
	return UpdateZoneCommandName
}

type UpdateZone struct {
	zr     ZoneRepository
	tracer trace.Tracer
}

func (u UpdateZone) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := u.tracer.Start(ctx, "UpdateZoneCmd")
	defer span.End()
	co, ok := cmd.(UpdateZoneCommand)
	if !ok {
		err := cqs.NewInvalidCommandError(UpdateZoneCommandName, cmd.Name())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err := u.update(ctx, co); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "zone updated")
	return nil, nil
}

func (u UpdateZone) update(ctx context.Context, co UpdateZoneCommand) error {
	relays := make([]zone.Relay, len(co.Relays))
	for i, re := range co.Relays {
		r, err := zone.ParseRelay(re)
		if err != nil {
			return UpdateZoneError{msg: err.Error()}
		}
		relays[i] = r
	}
	zo, err := u.zr.FindByID(ctx, co.ID)
	if err != nil {
		if errors.As(err, &errs.NotFoundError{}) {
			return UpdateZoneError{fmt.Sprintf("a zone with id %s, not found", co.ID)}
		}
		return err
	}
	if err = zo.Update(co.ZoneName, relays); err != nil {
		return UpdateZoneError{msg: err.Error()}
	}
	return u.zr.Update(ctx, zo)
}

func NewUpdateZone(zr ZoneRepository, tracer trace.Tracer) *UpdateZone {
	return &UpdateZone{zr: zr, tracer: tracer}
}

type UpdateZoneError struct {
	msg string
}

func (u UpdateZoneError) Error() string {
	return "failed to update zone: " + u.msg
}

package http

import (
	"errors"
	"net/http"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
)

func CreateZone(ch cqs.CommandHandler, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "CreateZoneRequest")
		defer span.End()
		var req Zone
		if err := ReadRequest(w, r, zoneRequestSchema, &req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}
		relays := make([]int, len(req.Relays))
		for i, r := range req.Relays {
			relays[i] = int(r)
		}
		if _, err := ch.Handle(ctx, app.CreateZoneCmd{
			ID:       req.Id,
			ZoneName: req.Name,
			Relays:   relays,
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			switch {
			case errors.As(err, &app.CreateZoneError{}):
				WriteErrorResponse(w, http.StatusBadRequest, Error{
					Code:    ErrorCodeInvalidRequest,
					Message: err.Error(),
				})
			default:
				WriteErrorResponse(w, http.StatusInternalServerError)
			}
			return
		}
		span.SetStatus(codes.Ok, "zone created")
		WriteResponse(w, http.StatusOK, nil)
	}
}

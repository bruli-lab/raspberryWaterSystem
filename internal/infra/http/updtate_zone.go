package http

import (
	"errors"
	"net/http"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func UpdateZone(ch cqs.CommandHandler, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "UpdateZoneRequest")
		defer span.End()
		id := chi.URLParam(r, "id")
		var req UpdateZoneRequest
		if err := ReadRequest(w, r, &req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}
		relays := make([]int, len(req.Relays))
		for i, relay := range req.Relays {
			relays[i] = int(relay)
		}
		if _, err := ch.Handle(ctx, app.UpdateZoneCommand{
			ID:       id,
			ZoneName: req.Name,
			Relays:   relays,
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			switch {
			case errors.As(err, &errs.NotFoundError{}):
				WriteErrorResponse(w, http.StatusNotFound)
			case errors.As(err, &app.UpdateZoneError{}):
				WriteErrorResponse(w, http.StatusBadRequest, Error{
					Code:    ErrorCodeInvalidRequest,
					Message: err.Error(),
				})
			default:
				WriteErrorResponse(w, http.StatusInternalServerError)
			}
			return
		}
		span.SetStatus(codes.Ok, "zone updated")
		WriteResponse(w, http.StatusOK, nil)
	}
}

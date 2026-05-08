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

func ExecuteFertilizerZone(ch cqs.CommandHandler, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "ExecuteFertilizerZoneRequest")
		defer span.End()
		var req ExecuteZoneRequestJson
		if err := ReadRequest(w, r, &req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}
		id := chi.URLParam(r, "id")
		if _, err := ch.Handle(ctx, app.ExecuteFertilizerZoneCmd{
			Seconds: uint(req.Seconds),
			ZoneID:  id,
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			switch {
			case errors.As(err, &errs.NotFoundError{}):
				WriteErrorResponse(w, http.StatusNotFound)
			default:
				WriteErrorResponse(w, http.StatusInternalServerError)
			}
			return
		}
		span.SetStatus(codes.Ok, "fertilizer zone executed")
		WriteResponse(w, http.StatusOK, nil)
	}
}

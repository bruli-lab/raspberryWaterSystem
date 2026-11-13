package http

import (
	"errors"
	"net/http"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func ExecuteCleanZone(ch cqs.CommandHandler, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "ExecuteCleanZoneRequest")
		defer span.End()
		id := chi.URLParam(r, "id")
		if _, err := ch.Handle(ctx, app.ExecuteCleanZoneCmd{
			ZoneID: id,
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			switch {
			case errors.As(err, &errs.NotFoundError{}):
				WriteErrorResponse(w, http.StatusNotFound)
			case errors.As(err, &app.CleanInProgressError{}):
				WriteErrorResponse(w, http.StatusConflict)
			default:
				WriteErrorResponse(w, http.StatusInternalServerError)
			}
			return
		}
		span.SetStatus(codes.Ok, "clean zone executed")
		WriteResponse(w, http.StatusOK, nil)
	}
}

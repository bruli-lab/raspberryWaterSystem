package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func RemoveTemperatureProgram(ch cqs.CommandHandler, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "RemoveTemperatureProgramRequest")
		defer span.End()
		temp, err := strconv.ParseFloat(chi.URLParam(r, "temperature"), 32)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			WriteErrorResponse(w, http.StatusBadRequest, Error{
				Code:    ErrorCodeInvalidRequest,
				Message: "invalid temperature value, must be a number",
			})
			return
		}
		if _, err = ch.Handle(ctx, app.RemoveTemperatureProgramCommand{Temperature: float32(temp)}); err != nil {
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
		span.SetStatus(codes.Ok, "temperature program removed")
		WriteResponse(w, http.StatusOK, nil)
	}
}

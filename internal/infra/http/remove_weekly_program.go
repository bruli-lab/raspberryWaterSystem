package http

import (
	"errors"
	"net/http"
	"unicode"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func RemoveWeeklyProgram(ch cqs.CommandHandler, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "RemoveWeeklyProgramRequest")
		defer span.End()
		day, err := program.ParseWeekDay(capitalizeDay(chi.URLParam(r, "day")))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			WriteErrorResponse(w, http.StatusBadRequest, Error{
				Code:    ErrorCodeInvalidRequest,
				Message: err.Error(),
			})
			return
		}
		if _, err = ch.Handle(ctx, app.RemoveWeeklyProgramCommand{Day: &day}); err != nil {
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
		span.SetStatus(codes.Ok, "weekly program removed")
		WriteResponse(w, http.StatusOK, nil)
	}
}

func capitalizeDay(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

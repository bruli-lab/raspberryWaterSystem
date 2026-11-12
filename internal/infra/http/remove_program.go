package http

import (
	"errors"
	"net/http"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func RemoveProgram(ch cqs.CommandHandler, prg string, tracer trace.Tracer) http.HandlerFunc {
	return func(writer http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "RemoveProgramRequest")
		defer span.End()
		hour, err := program.ParseHour(chi.URLParam(r, "hour"))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			WriteErrorResponse(writer, http.StatusBadRequest, Error{
				Code:    ErrorCodeInvalidRequest,
				Message: err.Error(),
			})
			return
		}
		if _, err := ch.Handle(ctx, buildRemoveProgramCommand(hour, prg)); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			switch {
			case errors.As(err, &errs.NotFoundError{}):
				WriteErrorResponse(writer, http.StatusNotFound)
			default:
				WriteErrorResponse(writer, http.StatusInternalServerError)
			}
		}
		span.SetStatus(codes.Ok, "program removed")
		WriteResponse(writer, http.StatusOK, nil)
	}
}

func buildRemoveProgramCommand(hour program.Hour, prg string) cqs.Command {
	switch prg {
	case DailyProgram:
		return app.RemoveDailyProgramCommand{
			Hour: &hour,
		}
	case OddProgram:
		return app.RemoveOddProgramCommand{
			Hour: &hour,
		}
	case EvenProgram:
		return app.RemoveEvenProgramCommand{
			Hour: &hour,
		}
	}
	return nil
}

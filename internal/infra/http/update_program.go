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

func UpdateProgram(ch cqs.CommandHandler, programType string, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "UpdateProgramRequest")
		defer span.End()
		var req UpdateProgramRequest
		if err := ReadRequest(w, r, updateProgramRequestSchema, &req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}
		in := app.UpdateProgramInput{
			Hour:       chi.URLParam(r, "hour"),
			Status:     programStatusInput(req.Status),
			Fertilizer: req.Fertilizer != nil && *req.Fertilizer,
			Executions: programExecutionInputs(req.Executions),
		}
		if _, err := ch.Handle(ctx, buildUpdateProgramCommand(programType, in)); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			switch {
			case errors.As(err, &errs.NotFoundError{}):
				WriteErrorResponse(w, http.StatusNotFound)
			case errors.As(err, &app.UpdateProgramError{}):
				WriteErrorResponse(w, http.StatusBadRequest, Error{
					Code:    ErrorCodeInvalidRequest,
					Message: err.Error(),
				})
			default:
				WriteErrorResponse(w, http.StatusInternalServerError)
			}
			return
		}
		span.SetStatus(codes.Ok, "program updated")
		WriteResponse(w, http.StatusOK, nil)
	}
}

func buildUpdateProgramCommand(programType string, in app.UpdateProgramInput) cqs.Command {
	switch programType {
	case DailyProgram:
		return app.UpdateDailyProgramCommand{UpdateProgramInput: in}
	case OddProgram:
		return app.UpdateOddProgramCommand{UpdateProgramInput: in}
	case EvenProgram:
		return app.UpdateEvenProgramCommand{UpdateProgramInput: in}
	default:
	}
	return nil
}

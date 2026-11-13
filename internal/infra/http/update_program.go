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

func UpdateProgram(ch cqs.CommandHandler, programType string, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "UpdateProgramRequest")
		defer span.End()
		hour, err := program.ParseHour(chi.URLParam(r, "hour"))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			WriteErrorResponse(w, http.StatusBadRequest, Error{
				Code:    ErrorCodeInvalidRequest,
				Message: err.Error(),
			})
			return
		}
		var req UpdateProgramRequest
		if err = ReadRequest(w, r, updateProgramRequestSchema, &req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}
		prog, err := buildUpdateProgram(w, hour, req)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}
		if _, err = ch.Handle(ctx, buildUpdateProgramCommand(programType, prog)); err != nil {
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

func buildUpdateProgramCommand(programType string, prog *program.Program) cqs.Command {
	switch programType {
	case DailyProgram:
		return app.UpdateDailyProgramCommand{Program: prog}
	case OddProgram:
		return app.UpdateOddProgramCommand{Program: prog}
	case EvenProgram:
		return app.UpdateEvenProgramCommand{Program: prog}
	default:
	}
	return nil
}

func buildUpdateProgram(w http.ResponseWriter, hour program.Hour, req UpdateProgramRequest) (*program.Program, error) {
	exec := make([]program.Execution, len(req.Executions))
	for i, ex := range req.Executions {
		sec, err := program.ParseSeconds(ex.Seconds)
		if err != nil {
			WriteErrorResponse(w, http.StatusBadRequest, Error{
				Code:    ErrorCodeInvalidRequest,
				Message: err.Error(),
			})
			return nil, err
		}
		e, err := program.NewExecution(sec, ex.Zones)
		if err != nil {
			WriteErrorResponse(w, http.StatusBadRequest, Error{
				Code:    ErrorCodeInvalidRequest,
				Message: err.Error(),
			})
			return nil, err
		}
		exec[i] = e
	}
	prog, err := program.New(hour, req.Fertilizer != nil && *req.Fertilizer, exec)
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, Error{
			Code:    ErrorCodeInvalidRequest,
			Message: err.Error(),
		})
		return nil, err
	}
	return prog, nil
}

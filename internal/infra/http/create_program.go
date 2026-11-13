package http

import (
	"errors"
	"net/http"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	DailyProgram = "daily"
	OddProgram   = "odd"
	EvenProgram  = "even"
)

func CreateProgram(ch cqs.CommandHandler, programType string, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "CreateProgramRequest")
		defer span.End()
		var req FertilizerProgram
		if err := ReadRequest(w, r, fertilizerProgramRequestSchema, &req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}
		prog, err := buildProgram(w, req)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}
		if _, err = ch.Handle(ctx, buildCreateProgramCommand(programType, prog)); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			switch {
			case errors.As(err, &app.CreateProgramError{}):
				WriteErrorResponse(w, http.StatusBadRequest, Error{
					Code:    ErrorCodeInvalidRequest,
					Message: err.Error(),
				})
				return
			default:
				WriteErrorResponse(w, http.StatusInternalServerError)
			}
			return
		}
		span.SetStatus(codes.Ok, "program created")
		WriteResponse(w, http.StatusOK, nil)
	}
}

func buildCreateProgramCommand(programType string, prog *program.Program) cqs.Command {
	switch programType {
	case DailyProgram:
		return app.CreateDailyProgramCommand{Program: prog}
	case OddProgram:
		return app.CreateOddProgramCommand{Program: prog}
	case EvenProgram:
		return app.CreateEvenProgramCommand{Program: prog}
	default:
	}
	return nil
}

func buildProgram(w http.ResponseWriter, req FertilizerProgram) (*program.Program, error) {
	hour, err := program.ParseHour(req.Hour)
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, Error{
			Code:    ErrorCodeInvalidRequest,
			Message: err.Error(),
		})
		return nil, err
	}
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

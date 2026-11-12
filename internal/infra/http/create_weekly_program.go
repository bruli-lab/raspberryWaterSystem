package http

import (
	"errors"
	"net/http"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func CreateWeeklyProgram(ch cqs.CommandHandler, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "CreateWeeklyProgramRequest")
		defer span.End()
		var req WeeklyProgramRequest
		if err := ReadRequest(w, r, weeklyProgramRequestSchema, &req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}
		cmd, err := buildCreateWeeklyProgramCommand(req)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			WriteErrorResponse(w, http.StatusBadRequest, Error{
				Code:    ErrorCodeInvalidRequest,
				Message: err.Error(),
			})
			return
		}
		if _, err := ch.Handle(ctx, app.CreateWeeklyProgramCommand{Weekly: cmd}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			switch {
			case errors.As(err, &app.CreateWeeklyProgramError{}):
				WriteErrorResponse(w, http.StatusBadRequest, Error{
					Code:    ErrorCodeInvalidRequest,
					Message: err.Error(),
				})
				return
			default:
				WriteErrorResponse(w, http.StatusInternalServerError, Error{})
				return
			}
		}
		span.SetStatus(codes.Ok, "weekly program created")
		WriteResponse(w, http.StatusOK, nil)
	}
}

func buildCreateWeeklyProgramCommand(req WeeklyProgramRequest) (*program.Weekly, error) {
	day, err := program.ParseWeekDay(req.WeekDay)
	if err != nil {
		return nil, err
	}
	prgms := make([]program.Program, len(req.Programs))
	for i, p := range req.Programs {
		hour, err := program.ParseHour(p.Hour)
		if err != nil {
			return nil, err
		}
		exec := make([]program.Execution, len(p.Executions))
		for j, e := range p.Executions {
			sec, err := program.ParseSeconds(e.Seconds)
			if err != nil {
				return nil, err
			}
			exe, err := program.NewExecution(sec, e.Zones)
			if err != nil {
				return nil, err
			}
			exec[j] = exe
		}
		pr, err := program.New(hour, exec)
		if err != nil {
			return nil, err
		}
		prgms[i] = *pr
	}
	weekly, err := program.NewWeekly(day, prgms)
	if err != nil {
		return nil, err
	}
	return &weekly, nil
}

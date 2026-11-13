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

func UpdateWeeklyProgram(ch cqs.CommandHandler, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "UpdateWeeklyProgramRequest")
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
		var req UpdateWeeklyProgramRequest
		if err = ReadRequest(w, r, updateWeeklyProgramRequestSchema, &req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}
		prgms, err := buildUpdateWeeklyPrograms(req)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			WriteErrorResponse(w, http.StatusBadRequest, Error{
				Code:    ErrorCodeInvalidRequest,
				Message: err.Error(),
			})
			return
		}
		if _, err = ch.Handle(ctx, app.UpdateWeeklyProgramCommand{WeekDay: day, Programs: prgms}); err != nil {
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
		span.SetStatus(codes.Ok, "weekly program updated")
		WriteResponse(w, http.StatusOK, nil)
	}
}

func buildUpdateWeeklyPrograms(req UpdateWeeklyProgramRequest) ([]program.Program, error) {
	prgms := make([]program.Program, len(req.Programs))
	for i, p := range req.Programs {
		hour, err := program.ParseHour(p.Hour)
		if err != nil {
			return nil, err
		}
		exec := make([]program.Execution, len(p.Executions))
		for n, ex := range p.Executions {
			sec, err := program.ParseSeconds(ex.Seconds)
			if err != nil {
				return nil, err
			}
			exe, err := program.NewExecution(sec, ex.Zones)
			if err != nil {
				return nil, err
			}
			exec[n] = exe
		}
		pr, err := program.NewWithoutFertilizer(hour, exec)
		if err != nil {
			return nil, err
		}
		prgms[i] = *pr
	}
	return prgms, nil
}

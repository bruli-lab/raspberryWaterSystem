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

func CreateTemperatureProgram(ch cqs.CommandHandler, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "CreateTemperatureProgramRequest")
		defer span.End()
		var req TemperatureProgramRequest
		if err := ReadRequest(w, r, temperatureProgramRequestSchema, &req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}
		cmd, err := buildCreateTemperatureProgram(req)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			WriteErrorResponse(w, http.StatusBadRequest, Error{
				Code:    ErrorCodeInvalidRequest,
				Message: err.Error(),
			})
			return
		}
		if _, err := ch.Handle(ctx, app.CreateTemperatureProgramCommand{
			Temperature: cmd,
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			switch {
			case errors.As(err, &app.CreateTemperatureProgramError{}):
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
		span.SetStatus(codes.Ok, "temperature program created")
		WriteResponse(w, http.StatusOK, nil)
	}
}

func buildCreateTemperatureProgram(req TemperatureProgramRequest) (*program.Temperature, error) {
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
	temp, err := program.NewTemperature(float32(req.Temperature), prgms)
	if err != nil {
		return nil, err
	}
	return temp, nil
}

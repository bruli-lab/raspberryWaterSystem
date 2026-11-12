package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func FindExecutionLogs(qh cqs.QueryHandler, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "FindExecutionLogsRequest")
		defer span.End()
		limit := 5
		limitStr := r.URL.Query().Get("limit")
		if limitStr != "" {
			limitValue, err := strconv.Atoi(limitStr)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				WriteErrorResponse(w, http.StatusBadRequest, Error{
					Code:    ErrorCodeInvalidRequest,
					Message: "invalid limit value",
				})
				return
			}
			limit = limitValue
		}
		result, err := qh.Handle(ctx, app.FindExecutionLogsQuery{Limit: limit})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			switch {
			case errors.Is(err, app.ErrInvalidExecutionsLogLimit):
				WriteErrorResponse(w, http.StatusBadRequest, Error{
					Code:    ErrorCodeInvalidRequest,
					Message: err.Error(),
				})
			default:
				WriteErrorResponse(w, http.StatusInternalServerError)
			}
			return
		}
		logs, _ := result.([]program.ExecutionLog)
		resp := make([]ExecutionLogsResponse, len(logs))
		for i, log := range logs {
			resp[i] = ExecutionLogsResponse{
				ExecutedAt: strconv.Itoa(int(log.ExecutedAt().Unix())),
				Seconds:    log.Seconds().Int(),
				ZoneName:   log.ZoneName(),
			}
		}
		data, _ := json.Marshal(resp)
		span.SetStatus(codes.Ok, "logs found")
		WriteResponse(w, http.StatusOK, data)
	}
}

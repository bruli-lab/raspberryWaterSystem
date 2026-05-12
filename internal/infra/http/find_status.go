package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func FindStatus(qh cqs.QueryHandler, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "FindStatusRequest")
		defer span.End()
		result, err := qh.Handle(ctx, app.FindStatusQuery{})
		if err != nil {
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
		currenSt, _ := result.(status.Status)
		var updated *string
		if currenSt.UpdatedAt() != nil {
			updated = new(strconv.Itoa(int(currenSt.UpdatedAt().Unix())))
		}
		resp := StatusResponse{
			Active:          currenSt.IsActive(),
			Humidity:        float32(currenSt.Weather().Humidity()),
			IsDay:           currenSt.Light().IsDay(time.Now()),
			IsRaining:       currenSt.Weather().IsRaining(),
			SystemStartedAt: strconv.Itoa(int(currenSt.SystemStartedAt().Unix())),
			Temperature:     float32(currenSt.Weather().Temperature()),
			UpdatedAt:       updated,
		}
		data, _ := json.Marshal(resp)
		span.SetStatus(codes.Ok, "OK")
		WriteResponse(w, http.StatusOK, data)
	}
}

package http

import (
	"encoding/json"
	"net/http"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/weather"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func FindWeather(qh cqs.QueryHandler, tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "FindWeatherRequest")
		defer span.End()
		result, err := qh.Handle(ctx, app.FindWeatherQuery{})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			WriteErrorResponse(w, http.StatusInternalServerError)
			return
		}
		weath, _ := result.(weather.Weather)
		resp := WeatherResponse{
			Humidity:    float32(weath.Humidity()),
			IsRaining:   weath.IsRaining(),
			Temperature: float32(weath.Temperature()),
		}
		data, _ := json.Marshal(resp)
		span.SetStatus(codes.Ok, "OK")
		WriteResponse(w, http.StatusOK, data)
	}
}

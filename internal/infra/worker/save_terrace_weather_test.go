package worker_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/weather"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"github.com/bruli/raspberryWaterSystem/internal/infra/worker"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

type eventsWriter struct {
	saved   []*outbox.Event
	saveErr error
}

func (e *eventsWriter) Save(_ context.Context, ev *outbox.Event) error {
	if e.saveErr != nil {
		return e.saveErr
	}
	e.saved = append(e.saved, ev)
	return nil
}

func TestSaveTerraceWeather(t *testing.T) {
	now := time.Date(2026, time.November, 13, 10, 0, 0, 0, time.UTC)
	temp := weather.Temperature(21.5)
	hum := weather.Humidity(60)
	weath := fixtures.WeatherBuilder{Temp: &temp, Humidity: &hum, Raining: true}.Build()
	st := fixtures.StatusBuilder{Weather: &weath}.Build()
	errTest := errors.New("test error")
	tests := []struct {
		name          string
		queryResult   any
		queryErr      error
		saveErr       error
		expectedErr   error
		expectedErrAs any
	}{
		{
			name:        "and find status query returns an error, then it returns same error",
			queryErr:    errTest,
			expectedErr: errTest,
		},
		{
			name:          "and find status query returns something that is not a status, then it returns an unexpected query result error",
			queryResult:   "not a status",
			expectedErrAs: &worker.UnexpectedQueryResultError{},
		},
		{
			name:        "and save returns an error, then it returns same error",
			queryResult: st,
			saveErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and everything works, then it saves a terrace weather event with the current weather",
			queryResult: st,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a SaveTerraceWeather worker, when it runs `+tt.name, func(t *testing.T) {
			t.Parallel()
			qh := &cqs.QueryHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Query) (any, error) {
					return tt.queryResult, tt.queryErr
				},
			}
			writer := &eventsWriter{saveErr: tt.saveErr}
			err := worker.SaveTerraceWeather(t.Context(), qh, writer, now, noop.NewTracerProvider().Tracer("test"))
			switch {
			case tt.expectedErr != nil:
				require.ErrorIs(t, err, tt.expectedErr)
				return
			case tt.expectedErrAs != nil:
				require.ErrorAs(t, err, tt.expectedErrAs)
				return
			}
			require.NoError(t, err)
			require.Len(t, writer.saved, 1)
			require.Equal(t, outbox.TerraceWeatherEventType, writer.saved[0].EventType)
			require.Equal(t, now, writer.saved[0].EventAt)
			var payload outbox.Weather
			require.NoError(t, json.Unmarshal(writer.saved[0].Payload, &payload))
			require.Equal(t, outbox.Weather{Temperature: 21.5, IsRaining: true, Humidity: 60}, payload)
		})
	}
}

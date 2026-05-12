package http_test

import (
	"context"
	"errors"
	"fmt"
	http2 "net/http"
	"net/http/httptest"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestUpdateTemperatureProgram(t *testing.T) {
	body := buildRequestJsonToString(t, http.UpdateTemperatureProgramRequest{
		Programs: []http.Program{
			{
				Executions: []http.ExecutionRequest{
					{
						Seconds: 10,
						Zones:   []string{"a", "b"},
					},
				},
				Hour: "10:00",
			},
		},
	})
	tests := []struct {
		name         string
		body         string
		ch           cqs.CommandHandler
		chErr        error
		expectedCode int
		temperature  string
	}{
		{
			name:         "with an invalid temperature, then it returns a bad request",
			temperature:  "invalid",
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:         "with an invalid request, then it returns a bad request",
			temperature:  "20",
			body:         "invalid",
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:        "and build programs returns error, then it returns a bad request",
			temperature: "20",
			body: buildRequestJsonToString(t, http.UpdateTemperatureProgramRequest{
				Programs: []http.Program{
					{
						Executions: []http.ExecutionRequest{
							{
								Seconds: 10,
								Zones:   []string{"a", "b"},
							},
						},
						Hour: "-10:00",
					},
				},
			}),
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:         "and command handler returns a not found error, then it returns a not found",
			temperature:  "20",
			body:         body,
			expectedCode: http2.StatusNotFound,
			chErr:        errs.NotFoundError{},
		},
		{
			name:         "and command handler returns an error, then it returns an internal server error",
			temperature:  "20",
			body:         body,
			expectedCode: http2.StatusInternalServerError,
			chErr:        errors.New(""),
		},
		{
			name:         "and command handler returns nil, then it returns an ok status",
			temperature:  "20",
			body:         body,
			expectedCode: http2.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an UpdateTemperateProgram http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			ch := &CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]cqs.Event, error) {
					return nil, tt.chErr
				},
			}
			handler := http.UpdateTemperatureProgram(ch, tracer())
			server := chi.NewMux()
			server.Put("/programs/temperature/{temperature}", handler)
			url := fmt.Sprintf("/programs/temperature/%s", tt.temperature)
			req := httptest.NewRequest(http2.MethodPut, url, buildRequestBody(tt.body))
			writer := httptest.NewRecorder()
			server.ServeHTTP(writer, req)
			resp := writer.Result()
			require.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}

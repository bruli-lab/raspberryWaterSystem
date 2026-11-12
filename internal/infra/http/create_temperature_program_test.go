package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	http2 "github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/stretchr/testify/require"
)

func TestCreateTemperatureProgram(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		expectedCode int
		chErr        error
	}{
		{
			name:         "with an invalid request, then it returns a bad request",
			body:         "invalid",
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "with an invalid hour in the request, then it returns a bad request",
			body: buildRequestJsonToString(t, http2.TemperatureProgramRequest{
				Programs: []http2.Program{
					{
						Executions: []http2.ExecutionRequest{
							{
								Seconds: 20,
								Zones:   []string{"a", "b"},
							},
						},
						Hour: "invalid",
					},
				},
				Temperature: 20,
			}),
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "with an invalid seconds in the request, then it returns a bad request",
			body: buildRequestJsonToString(t, http2.TemperatureProgramRequest{
				Programs: []http2.Program{
					{
						Executions: []http2.ExecutionRequest{
							{
								Seconds: -1,
								Zones:   []string{"a", "b"},
							},
						},
						Hour: "10:45",
					},
				},
				Temperature: 20,
			}),
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "and command handler returns a create weekly program error, then it returns a bad request",
			body: buildRequestJsonToString(t, http2.TemperatureProgramRequest{
				Programs: []http2.Program{
					{
						Executions: []http2.ExecutionRequest{
							{
								Seconds: 10,
								Zones:   []string{"a"},
							},
						},
						Hour: "10:45",
					},
				},
				Temperature: 20,
			}),
			chErr:        app.CreateTemperatureProgramError{},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "and command handler returns an error, then it returns an internal server error",
			body: buildRequestJsonToString(t, http2.TemperatureProgramRequest{
				Programs: []http2.Program{
					{
						Executions: []http2.ExecutionRequest{
							{
								Seconds: 10,
								Zones:   []string{"a"},
							},
						},
						Hour: "10:45",
					},
				},
				Temperature: 20,
			}),
			chErr:        errors.New(""),
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "and command handler returns nil, then it returns an ok status",
			body: buildRequestJsonToString(t, http2.TemperatureProgramRequest{
				Programs: []http2.Program{
					{
						Executions: []http2.ExecutionRequest{
							{
								Seconds: 10,
								Zones:   []string{"a"},
							},
						},
						Hour: "10:45",
					},
				},
				Temperature: 20,
			}),
			expectedCode: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a CreateTemperatureProgram http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			ch := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
					return nil, tt.chErr
				},
			}
			handler := http2.CreateTemperatureProgram(ch, tracer())
			req := httptest.NewRequest(http.MethodPost, "/programs/temperature", buildRequestBody(tt.body))
			writer := httptest.NewRecorder()
			handler.ServeHTTP(writer, req)
			resp := writer.Result()
			require.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}

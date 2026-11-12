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

func TestCreateProgram(t *testing.T) {
	hour := "15:30"
	zones := []string{"1", "2"}
	exec := []http2.ExecutionRequest{
		{
			Seconds: 20,
			Zones:   zones,
		},
	}
	body := buildRequestJsonToString(t, http2.Program{
		Executions: []http2.ExecutionRequest{
			{
				Seconds: 20,
				Zones:   []string{"1", "2"},
			},
		},
		Hour: "15:30",
	})
	tests := []struct {
		name         string
		chErr        error
		body         string
		expectedCode int
	}{
		{
			name:         "with an invalid request, then it returns a bad request",
			body:         "invalid",
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "with an invalid hour in request, then it returns a bad request",
			body: buildRequestJsonToString(t, http2.Program{
				Executions: exec,
				Hour:       "invalid",
			}),
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "with an invalid seconds in request, then it returns a bad request",
			body: buildRequestJsonToString(t, http2.Program{
				Executions: []http2.ExecutionRequest{
					{
						Seconds: -10,
						Zones:   zones,
					},
				},
				Hour: hour,
			}),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "with a valid request and command handler returns a create program error, then it returns a bad request",
			body:         body,
			chErr:        app.CreateProgramError{},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "with a valid request and command handler returns an error, then it returns an internal server error",
			body:         body,
			chErr:        errors.New(""),
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "with a valid request and command handler returns nil, then it returns an valid response code",
			body:         body,
			expectedCode: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a CreateProgram http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			ch := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
					return nil, tt.chErr
				},
			}
			handler := http2.CreateProgram(ch, http2.DailyProgram, tracer())
			req := httptest.NewRequest(http.MethodPost, "/programs", buildRequestBody(tt.body))
			writer := httptest.NewRecorder()
			handler.ServeHTTP(writer, req)
			resp := writer.Result()
			require.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}

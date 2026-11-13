package http_test

import (
	"context"
	"errors"
	http2 "net/http"
	"net/http/httptest"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestUpdateWeeklyProgram(t *testing.T) {
	body := buildRequestJsonToString(t, http.UpdateWeeklyProgramRequest{
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
		chErr        error
		expectedCode int
		day          string
	}{
		{
			name:         "with an invalid day, then it returns a bad request",
			day:          "invalid",
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:         "with an invalid request, then it returns a bad request",
			day:          "monday",
			body:         "invalid",
			expectedCode: http2.StatusBadRequest,
		},
		{
			name: "and build programs returns an error, then it returns a bad request",
			day:  "monday",
			body: buildRequestJsonToString(t, http.UpdateWeeklyProgramRequest{
				Programs: []http.Program{
					{
						Executions: []http.ExecutionRequest{
							{Seconds: 10, Zones: []string{"a"}},
						},
						Hour: "-10:00",
					},
				},
			}),
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:         "and command handler returns a not found error, then it returns a not found",
			day:          "monday",
			body:         body,
			chErr:        errs.NotFoundError{},
			expectedCode: http2.StatusNotFound,
		},
		{
			name:         "and command handler returns an error, then it returns an internal server error",
			day:          "monday",
			body:         body,
			chErr:        errors.New(""),
			expectedCode: http2.StatusInternalServerError,
		},
		{
			name:         "and command handler returns nil, then it returns an ok status",
			day:          "monday",
			body:         body,
			expectedCode: http2.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an UpdateWeeklyProgram http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			ch := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
					return nil, tt.chErr
				},
			}
			handler := http.UpdateWeeklyProgram(ch, tracer())
			server := chi.NewMux()
			server.Put("/programs/weekly/{day}", handler)
			req := httptest.NewRequestWithContext(t.Context(), http2.MethodPut, "/programs/weekly/"+tt.day, buildRequestBody(tt.body))
			writer := httptest.NewRecorder()

			server.ServeHTTP(writer, req)

			require.Equal(t, tt.expectedCode, writer.Result().StatusCode)
		})
	}
}

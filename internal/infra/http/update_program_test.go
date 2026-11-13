package http_test

import (
	"context"
	"errors"
	http2 "net/http"
	"net/http/httptest"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestUpdateProgram(t *testing.T) {
	hour, err := program.ParseHour("10:15")
	require.NoError(t, err)
	seconds, err := program.ParseSeconds(20)
	require.NoError(t, err)
	execution, err := program.NewExecution(seconds, []string{"1", "2"})
	require.NoError(t, err)
	prog, err := program.New(hour, true, []program.Execution{execution})
	require.NoError(t, err)
	body := buildRequestJsonToString(t, http.UpdateProgramRequest{
		Executions: []http.ExecutionRequest{{Seconds: 20, Zones: []string{"1", "2"}}},
		Fertilizer: new(true),
	})
	tests := []struct {
		name            string
		programType     string
		hour            string
		body            string
		chErr           error
		expectedCode    int
		expectedCommand cqs.Command
	}{
		{
			name:         "with an invalid hour, then it returns a bad request",
			programType:  http.DailyProgram,
			hour:         "invalid",
			body:         body,
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:         "with an invalid request body, then it returns a bad request",
			programType:  http.DailyProgram,
			hour:         "10:15",
			body:         "invalid",
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:        "with an invalid seconds in an execution, then it returns a bad request",
			programType: http.DailyProgram,
			hour:        "10:15",
			body: buildRequestJsonToString(t, http.UpdateProgramRequest{
				Executions: []http.ExecutionRequest{{Seconds: -10, Zones: []string{"1"}}},
			}),
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:            "and command handler returns a not found error, then it returns a not found",
			programType:     http.DailyProgram,
			hour:            "10:15",
			body:            body,
			chErr:           errs.NotFoundError{},
			expectedCode:    http2.StatusNotFound,
			expectedCommand: app.UpdateDailyProgramCommand{Program: prog},
		},
		{
			name:            "and command handler returns an update program error, then it returns a bad request",
			programType:     http.OddProgram,
			hour:            "10:15",
			body:            body,
			chErr:           app.UpdateProgramError{},
			expectedCode:    http2.StatusBadRequest,
			expectedCommand: app.UpdateOddProgramCommand{Program: prog},
		},
		{
			name:            "and command handler returns an error, then it returns an internal server error",
			programType:     http.EvenProgram,
			hour:            "10:15",
			body:            body,
			chErr:           errors.New(""),
			expectedCode:    http2.StatusInternalServerError,
			expectedCommand: app.UpdateEvenProgramCommand{Program: prog},
		},
		{
			name:            "for a daily program and command handler returns nil, then it updates the daily program",
			programType:     http.DailyProgram,
			hour:            "10:15",
			body:            body,
			expectedCode:    http2.StatusOK,
			expectedCommand: app.UpdateDailyProgramCommand{Program: prog},
		},
		{
			name:            "for an odd program and command handler returns nil, then it updates the odd program",
			programType:     http.OddProgram,
			hour:            "10:15",
			body:            body,
			expectedCode:    http2.StatusOK,
			expectedCommand: app.UpdateOddProgramCommand{Program: prog},
		},
		{
			name:            "for an even program and command handler returns nil, then it updates the even program",
			programType:     http.EvenProgram,
			hour:            "10:15",
			body:            body,
			expectedCode:    http2.StatusOK,
			expectedCommand: app.UpdateEvenProgramCommand{Program: prog},
		},
	}
	for _, tt := range tests {
		t.Run(`Given an UpdateProgram http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			ch := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
					return nil, tt.chErr
				},
			}
			server := chi.NewMux()
			server.Put("/programs/{hour}", http.UpdateProgram(ch, tt.programType, tracer()))
			req := httptest.NewRequestWithContext(t.Context(), http2.MethodPut, "/programs/"+tt.hour, buildRequestBody(tt.body))
			writer := httptest.NewRecorder()

			server.ServeHTTP(writer, req)

			require.Equal(t, tt.expectedCode, writer.Result().StatusCode)
			if tt.expectedCommand == nil {
				require.Empty(t, ch.HandleCalls())
				return
			}
			require.Len(t, ch.HandleCalls(), 1)
			require.Equal(t, tt.expectedCommand, ch.HandleCalls()[0].Cmd)
		})
	}
}

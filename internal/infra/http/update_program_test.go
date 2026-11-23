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
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestUpdateProgram(t *testing.T) {
	disabled := http.Disabled
	body := buildRequestJsonToString(t, http.UpdateProgramRequest{
		Executions: []http.ExecutionRequest{{Seconds: 20, Zones: []string{"1", "2"}}},
		Fertilizer: new(true),
		Status:     &disabled,
	})
	input := app.UpdateProgramInput{
		Hour:       "10:15",
		Status:     "disabled",
		Fertilizer: true,
		Executions: []app.ProgramExecutionInput{{Seconds: 20, Zones: []string{"1", "2"}}},
	}
	withoutStatusInput := app.UpdateProgramInput{
		Hour:       "10:15",
		Executions: []app.ProgramExecutionInput{{Seconds: 20, Zones: []string{"1"}}},
	}
	tests := []struct {
		name            string
		programType     string
		body            string
		chErr           error
		expectedCode    int
		expectedCommand cqs.Command
	}{
		{
			name:         "with an invalid request body, then it returns a bad request",
			programType:  http.DailyProgram,
			body:         "invalid",
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:         "with an unknown status, then it returns a bad request",
			programType:  http.DailyProgram,
			body:         `{"executions":[{"seconds":20,"zones":["1"]}],"status":"paused"}`,
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:            "and command handler returns a not found error, then it returns a not found",
			programType:     http.DailyProgram,
			body:            body,
			chErr:           errs.NotFoundError{},
			expectedCode:    http2.StatusNotFound,
			expectedCommand: app.UpdateDailyProgramCommand{UpdateProgramInput: input},
		},
		{
			name:            "and command handler returns an update program error, then it returns a bad request",
			programType:     http.OddProgram,
			body:            body,
			chErr:           app.UpdateProgramError{},
			expectedCode:    http2.StatusBadRequest,
			expectedCommand: app.UpdateOddProgramCommand{UpdateProgramInput: input},
		},
		{
			name:            "and command handler returns an error, then it returns an internal server error",
			programType:     http.EvenProgram,
			body:            body,
			chErr:           errors.New(""),
			expectedCode:    http2.StatusInternalServerError,
			expectedCommand: app.UpdateEvenProgramCommand{UpdateProgramInput: input},
		},
		{
			name:            "for a daily program and command handler returns nil, then it sends the request data to the command",
			programType:     http.DailyProgram,
			body:            body,
			expectedCode:    http2.StatusOK,
			expectedCommand: app.UpdateDailyProgramCommand{UpdateProgramInput: input},
		},
		{
			name:        "without status and fertilizer, then it sends them empty to the command",
			programType: http.DailyProgram,
			body: buildRequestJsonToString(t, http.UpdateProgramRequest{
				Executions: []http.ExecutionRequest{{Seconds: 20, Zones: []string{"1"}}},
			}),
			expectedCode:    http2.StatusOK,
			expectedCommand: app.UpdateDailyProgramCommand{UpdateProgramInput: withoutStatusInput},
		},
		{
			name:            "for an odd program and command handler returns nil, then it updates the odd program",
			programType:     http.OddProgram,
			body:            body,
			expectedCode:    http2.StatusOK,
			expectedCommand: app.UpdateOddProgramCommand{UpdateProgramInput: input},
		},
		{
			name:            "for an even program and command handler returns nil, then it updates the even program",
			programType:     http.EvenProgram,
			body:            body,
			expectedCode:    http2.StatusOK,
			expectedCommand: app.UpdateEvenProgramCommand{UpdateProgramInput: input},
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
			req := httptest.NewRequestWithContext(t.Context(), http2.MethodPut, "/programs/10:15", buildRequestBody(tt.body))
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

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

func updateProgramItems() []http.UpdateProgramItem {
	disabled := http.Disabled
	return []http.UpdateProgramItem{
		{Executions: []http.ExecutionRequest{{Seconds: 10, Zones: []string{"a", "b"}}}, Hour: "10:00", Status: &disabled},
		{Executions: []http.ExecutionRequest{{Seconds: 20, Zones: []string{"c"}}}, Hour: "11:00"},
	}
}

func expectedProgramInputs() []app.ProgramInput {
	return []app.ProgramInput{
		{Hour: "10:00", Status: "disabled", Executions: []app.ProgramExecutionInput{{Seconds: 10, Zones: []string{"a", "b"}}}},
		{Hour: "11:00", Executions: []app.ProgramExecutionInput{{Seconds: 20, Zones: []string{"c"}}}},
	}
}

func TestUpdateWeeklyProgram(t *testing.T) {
	body := buildRequestJsonToString(t, http.UpdateWeeklyProgramRequest{Programs: updateProgramItems()})
	expectedCommand := app.UpdateWeeklyProgramCommand{WeekDay: "Monday", Programs: expectedProgramInputs()}
	tests := []struct {
		name            string
		body            string
		chErr           error
		expectedCode    int
		expectedCommand cqs.Command
	}{
		{
			name:         "with an invalid request, then it returns a bad request",
			body:         "invalid",
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:            "and command handler returns an update weekly program error, then it returns a bad request",
			body:            body,
			chErr:           app.UpdateWeeklyProgramError{},
			expectedCode:    http2.StatusBadRequest,
			expectedCommand: expectedCommand,
		},
		{
			name:            "and command handler returns a not found error, then it returns a not found",
			body:            body,
			chErr:           errs.NotFoundError{},
			expectedCode:    http2.StatusNotFound,
			expectedCommand: expectedCommand,
		},
		{
			name:            "and command handler returns an error, then it returns an internal server error",
			body:            body,
			chErr:           errors.New(""),
			expectedCode:    http2.StatusInternalServerError,
			expectedCommand: expectedCommand,
		},
		{
			name:            "and command handler returns nil, then it sends the capitalized day and the programs to the command",
			body:            body,
			expectedCode:    http2.StatusOK,
			expectedCommand: expectedCommand,
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
			server := chi.NewMux()
			server.Put("/programs/weekly/{day}", http.UpdateWeeklyProgram(ch, tracer()))
			req := httptest.NewRequestWithContext(t.Context(), http2.MethodPut, "/programs/weekly/monday", buildRequestBody(tt.body))
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

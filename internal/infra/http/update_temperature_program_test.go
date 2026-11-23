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

func TestUpdateTemperatureProgram(t *testing.T) {
	body := buildRequestJsonToString(t, http.UpdateTemperatureProgramRequest{Programs: updateProgramItems()})
	expectedCommand := app.UpdateTemperatureProgramCommand{Temperature: 20.5, Programs: expectedProgramInputs()}
	tests := []struct {
		name, temperature, body string
		chErr                   error
		expectedCode            int
		expectedCommand         cqs.Command
	}{
		{
			name:         "with an invalid temperature, then it returns a bad request",
			temperature:  "invalid",
			body:         body,
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:         "with an invalid request, then it returns a bad request",
			temperature:  "20.5",
			body:         "invalid",
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:            "and command handler returns an update temperature program error, then it returns a bad request",
			temperature:     "20.5",
			body:            body,
			chErr:           app.UpdateTemperatureProgramError{},
			expectedCode:    http2.StatusBadRequest,
			expectedCommand: expectedCommand,
		},
		{
			name:            "and command handler returns a not found error, then it returns a not found",
			temperature:     "20.5",
			body:            body,
			chErr:           errs.NotFoundError{},
			expectedCode:    http2.StatusNotFound,
			expectedCommand: expectedCommand,
		},
		{
			name:            "and command handler returns an error, then it returns an internal server error",
			temperature:     "20.5",
			body:            body,
			chErr:           errors.New(""),
			expectedCode:    http2.StatusInternalServerError,
			expectedCommand: expectedCommand,
		},
		{
			name:            "and command handler returns nil, then it sends the temperature and the programs to the command",
			temperature:     "20.5",
			body:            body,
			expectedCode:    http2.StatusOK,
			expectedCommand: expectedCommand,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an UpdateTemperatureProgram http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			ch := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
					return nil, tt.chErr
				},
			}
			server := chi.NewMux()
			server.Put("/programs/temperature/{temperature}", http.UpdateTemperatureProgram(ch, tracer()))
			req := httptest.NewRequestWithContext(t.Context(), http2.MethodPut,
				"/programs/temperature/"+tt.temperature, buildRequestBody(tt.body))
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

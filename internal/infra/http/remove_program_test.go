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

func TestRemoveProgram(t *testing.T) {
	hour, err := program.ParseHour("10:15")
	require.NoError(t, err)
	tests := []struct {
		name            string
		program         string
		hour            string
		chErr           error
		expectedCode    int
		expectedCommand cqs.Command
	}{
		{
			name:         "with an invalid hour, then it returns a bad request",
			program:      http.DailyProgram,
			hour:         "invalid",
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:            "and command handler returns a not found error, then it returns a not found",
			program:         http.DailyProgram,
			hour:            "10:15",
			chErr:           errs.NotFoundError{},
			expectedCode:    http2.StatusNotFound,
			expectedCommand: app.RemoveDailyProgramCommand{Hour: &hour},
		},
		{
			name:            "and command handler returns an error, then it returns an internal server error",
			program:         http.OddProgram,
			hour:            "10:15",
			chErr:           errors.New(""),
			expectedCode:    http2.StatusInternalServerError,
			expectedCommand: app.RemoveOddProgramCommand{Hour: &hour},
		},
		{
			name:            "for a daily program and command handler returns nil, then it removes the daily program",
			program:         http.DailyProgram,
			hour:            "10:15",
			expectedCode:    http2.StatusOK,
			expectedCommand: app.RemoveDailyProgramCommand{Hour: &hour},
		},
		{
			name:            "for an odd program and command handler returns nil, then it removes the odd program",
			program:         http.OddProgram,
			hour:            "10:15",
			expectedCode:    http2.StatusOK,
			expectedCommand: app.RemoveOddProgramCommand{Hour: &hour},
		},
		{
			name:            "for an even program and command handler returns nil, then it removes the even program",
			program:         http.EvenProgram,
			hour:            "10:15",
			expectedCode:    http2.StatusOK,
			expectedCommand: app.RemoveEvenProgramCommand{Hour: &hour},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a RemoveProgram http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			ch := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
					return nil, tt.chErr
				},
			}
			server := chi.NewMux()
			server.Delete("/programs/{hour}", http.RemoveProgram(ch, tt.program, tracer()))
			req := httptest.NewRequestWithContext(t.Context(), http2.MethodDelete, "/programs/"+tt.hour, http2.NoBody)
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

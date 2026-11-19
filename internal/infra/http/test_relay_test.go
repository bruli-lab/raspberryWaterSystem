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
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
)

func TestTestRelay(t *testing.T) {
	tests := []struct {
		name, id     string
		chErr        error
		expectedCode int
		expectedCmd  cqs.Command
	}{
		{
			name:         "with a non numeric relay id, then it returns bad request without handling the command",
			id:           "pump",
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:         "and command handler returns an unknown relay error, then it returns bad request",
			id:           "99",
			chErr:        zone.ErrUnknownRelay,
			expectedCode: http2.StatusBadRequest,
			expectedCmd:  app.TestRelayCmd{RelayID: 99},
		},
		{
			name:         "and command handler returns a clean in progress error, then it returns conflict",
			id:           "1",
			chErr:        app.CleanInProgressError{},
			expectedCode: http2.StatusConflict,
			expectedCmd:  app.TestRelayCmd{RelayID: 1},
		},
		{
			name:         "and command handler returns an error, then it returns internal server error",
			id:           "1",
			chErr:        errors.New(""),
			expectedCode: http2.StatusInternalServerError,
			expectedCmd:  app.TestRelayCmd{RelayID: 1},
		},
		{
			name:         "with a valid relay id, then it returns ok",
			id:           "5",
			expectedCode: http2.StatusOK,
			expectedCmd:  app.TestRelayCmd{RelayID: 5},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a TestRelay http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			ch := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
					return nil, tt.chErr
				},
			}
			server := chi.NewMux()
			server.Post("/relays/{id}/test", http.TestRelay(ch, tracer()))
			req := httptest.NewRequestWithContext(t.Context(), http2.MethodPost, "/relays/"+tt.id+"/test", http2.NoBody)
			writer := httptest.NewRecorder()
			server.ServeHTTP(writer, req)
			require.Equal(t, tt.expectedCode, writer.Result().StatusCode)
			if tt.expectedCmd == nil {
				require.Empty(t, ch.HandleCalls())
				return
			}
			require.Len(t, ch.HandleCalls(), 1)
			require.Equal(t, tt.expectedCmd, ch.HandleCalls()[0].Cmd)
		})
	}
}

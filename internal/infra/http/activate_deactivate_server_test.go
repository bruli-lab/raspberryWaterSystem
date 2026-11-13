package http_test

import (
	"context"
	"errors"
	http2 "net/http"
	"net/http/httptest"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestActivateDeactivateServer(t *testing.T) {
	tests := []struct {
		name         string
		chErr        error
		expectedCode int
		action       string
	}{
		{
			name:         "with an invalid action name, then it returns a bad request",
			action:       "invalid",
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:         "with a valid action name and command handler returns error, then it returns an internal server error",
			action:       http.ActivateAction,
			chErr:        errors.New(""),
			expectedCode: http2.StatusInternalServerError,
		},
		{
			name:         "with a valid action name and command handler returns nil, then it returns ok",
			action:       http.DeactivateAction,
			expectedCode: http2.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an ActivateDeactivateServer http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			ch := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
					return nil, tt.chErr
				},
			}
			handler := http.ActivateDeactivateServer(ch, tracer())
			server := chi.NewMux()
			server.Patch("/status/{action}", handler)
			req := httptest.NewRequestWithContext(t.Context(), http2.MethodPatch, "/status/"+tt.action, http2.NoBody)
			writer := httptest.NewRecorder()
			server.ServeHTTP(writer, req)
			require.Equal(t, tt.expectedCode, writer.Code)
		})
	}
}

func tracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test")
}

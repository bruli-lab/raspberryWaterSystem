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
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
)

func TestExecuteCleanZone(t *testing.T) {
	tests := []struct {
		name         string
		chErr        error
		expectedCode int
	}{
		{
			name:         "and command handler returns not found error, then it returns not found",
			expectedCode: http2.StatusNotFound,
			chErr:        errs.NotFoundError{},
		},
		{
			name:         "and command handler returns a clean in progress error, then it returns conflict",
			expectedCode: http2.StatusConflict,
			chErr:        app.CleanInProgressError{},
		},
		{
			name:         "with a valid request and command handler returns en error, then it returns internal server error",
			expectedCode: http2.StatusInternalServerError,
			chErr:        errors.New(""),
		},
		{
			name:         "with a valid request, then it returns ok",
			expectedCode: http2.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a ExecuteCleanZone http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			ch := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
					return nil, tt.chErr
				},
			}
			handler := http.ExecuteCleanZone(ch, tracer())
			server := chi.NewMux()
			server.Post("/zones/clean/{id}/execute", handler)
			req := httptest.NewRequestWithContext(t.Context(), http2.MethodPost, "/zones/clean/id/execute", http2.NoBody)
			writer := httptest.NewRecorder()
			server.ServeHTTP(writer, req)
			resp := writer.Result()
			require.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}

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
	"github.com/stretchr/testify/require"
)

func TestUpdateZone(t *testing.T) {
	body := buildRequestJsonToString(t, http.UpdateZoneRequest{
		Name:   "test",
		Relays: []http.UpdateZoneRequestRelays{1},
	})
	tests := []struct {
		name         string
		body         string
		chErr        error
		expectedCode int
	}{
		{
			name:         "with an invalid request, then it returns a bad request",
			body:         "invalid",
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:         "and command handler returns a not found error, then it returns a not found",
			body:         body,
			chErr:        errs.NotFoundError{},
			expectedCode: http2.StatusNotFound,
		},
		{
			name:         "and command handler returns an update zone error, then it returns a bad request",
			body:         body,
			chErr:        app.UpdateZoneError{},
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:         "and command handler returns an update zone error, then it returns an internal server error",
			body:         body,
			chErr:        errors.New(""),
			expectedCode: http2.StatusInternalServerError,
		},
		{
			name:         "and command handler returns nil, then it returns an ok status",
			body:         body,
			expectedCode: http2.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a UpdateZone http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			ch := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
					return nil, tt.chErr
				},
			}
			handler := http.UpdateZone(ch, tracer())
			req := httptest.NewRequestWithContext(t.Context(), http2.MethodPut, "/zones/id", buildRequestBody(tt.body))
			writer := httptest.NewRecorder()
			handler.ServeHTTP(writer, req)
			resp := writer.Result()
			require.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}

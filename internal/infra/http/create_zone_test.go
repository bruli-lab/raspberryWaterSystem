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
	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/stretchr/testify/require"
)

func TestCreateZone(t *testing.T) {
	errTest := errors.New("")
	body := buildRequestJsonToString(t, http.Zone{
		Id:     "id",
		Name:   "name",
		Relays: []http.ZoneRelays{1},
	})
	tests := []struct {
		name, body   string
		expectedCode int
		chErr        error
	}{
		{
			name:         "with an invalid request, then it returns a bad request",
			body:         "invalid",
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:         "with a valid request and command handler returns a create zone error, then it returns a bad request",
			body:         body,
			chErr:        app.CreateZoneError{},
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:         "with a valid request and command handler returns an error, then it returns an internal server error",
			body:         body,
			chErr:        errTest,
			expectedCode: http2.StatusInternalServerError,
		},
		{
			name:         "with a valid request and command handler returns an error, then it returns an internal server error",
			body:         body,
			expectedCode: http2.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a CreateZone http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			ch := &cqs.CommandHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
					return nil, tt.chErr
				},
			}
			handler := http.CreateZone(ch, tracer())
			req := httptest.NewRequest(http2.MethodPost, "/zones", buildRequestBody(tt.body))
			writer := httptest.NewRecorder()
			handler.ServeHTTP(writer, req)
			resp := writer.Result()
			require.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}

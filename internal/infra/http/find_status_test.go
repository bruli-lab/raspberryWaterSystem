package http_test

import (
	"context"
	"errors"
	http2 "net/http"
	"net/http/httptest"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	errs "github.com/bruli/raspberryWaterSystem/internal/errors"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"

	"github.com/stretchr/testify/require"

	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
)

func TestFindStatus(t *testing.T) {
	tests := []struct {
		name         string
		expectedCode int
		result       any
		qhErr        error
	}{
		{
			name:         "and query handler returns a not found error, then it returns a not found",
			qhErr:        errs.NotFoundError{},
			expectedCode: http2.StatusNotFound,
		},
		{
			name:         "and query handler returns an error, then it returns an internal server error",
			qhErr:        errors.New(""),
			expectedCode: http2.StatusInternalServerError,
		},
		{
			name:         "and query handler returns an result, then it returns a valid response",
			result:       fixtures.StatusBuilder{}.Build(),
			expectedCode: http2.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a FindStatus query handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			qh := &cqs.QueryHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Query) (any, error) {
					return tt.result, tt.qhErr
				},
			}
			handler := http.FindStatus(qh, tracer())
			req := httptest.NewRequestWithContext(t.Context(), http2.MethodGet, "/status", http2.NoBody)
			writer := httptest.NewRecorder()
			handler.ServeHTTP(writer, req)
			resp := writer.Result()
			require.Equal(t, tt.expectedCode, resp.StatusCode)
			if resp.StatusCode == http2.StatusOK {
				var schema http.StatusResponse
				readResponse(t, resp, &schema)
			}
		})
	}
}

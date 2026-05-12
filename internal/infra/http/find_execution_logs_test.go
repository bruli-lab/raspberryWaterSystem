package http_test

import (
	"context"
	"errors"
	http2 "net/http"
	"net/http/httptest"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"

	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/stretchr/testify/require"
)

func TestFindExecutionLogs(t *testing.T) {
	tests := []struct {
		name, url    string
		expectedCode int
		qhErr        error
		result       any
	}{
		{
			name:         "with an invalid limit value, then it returns a bad request",
			url:          "/logs?limit=invalid",
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:         "and query handler returns an invalid execution log limit error, then it returns a bad request",
			url:          "/logs",
			qhErr:        app.ErrInvalidExecutionsLogLimit,
			expectedCode: http2.StatusBadRequest,
		},
		{
			name:         "and query handler returns an error, then it returns an internal server error",
			url:          "/logs",
			qhErr:        errors.New(""),
			expectedCode: http2.StatusInternalServerError,
		},
		{
			name:         "and query handler returns an valid result, then it returns a response",
			url:          "/logs?limit=2",
			expectedCode: http2.StatusOK,
			result: []program.ExecutionLog{
				fixtures.ExecutionLogBuilder{}.Build(),
				fixtures.ExecutionLogBuilder{}.Build(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a FindExecutionLogs http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			qh := &QueryHandlerMock{
				HandleFunc: func(_ context.Context, _ cqs.Query) (any, error) {
					return tt.result, tt.qhErr
				},
			}
			handler := http.FindExecutionLogs(qh, tracer())
			req := httptest.NewRequest(http2.MethodGet, tt.url, http2.NoBody)
			writer := httptest.NewRecorder()
			handler.ServeHTTP(writer, req)
			resp := writer.Result()
			require.Equal(t, tt.expectedCode, resp.StatusCode)
			if resp.StatusCode == http2.StatusOK {
				var schema []http.ExecutionLogsResponse
				readResponse(t, resp, &schema)
				require.Len(t, schema, 2)
			}
		})
	}
}

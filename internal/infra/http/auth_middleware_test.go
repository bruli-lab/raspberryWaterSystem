package http_test

import (
	http2 "net/http"
	"net/http/httptest"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name, token  string
		expectedCode int
	}{
		{
			name:         "without Authorization token, then it returns an unauthorized",
			expectedCode: http2.StatusUnauthorized,
		},
		{
			name:         "with invalid Authorization token, then it returns an unauthorized",
			token:        "invalid",
			expectedCode: http2.StatusUnauthorized,
		},
		{
			name:         "with valid Authorization token, then it returns an ok",
			token:        "1234",
			expectedCode: http2.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an AuthMiddleware,
		when is used and a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			middleware := http.AuthMiddleware("1234")
			handler := middleware(nextHandler())
			req := httptest.NewRequestWithContext(t.Context(), http2.MethodPost, "/", http2.NoBody)
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}
			writer := httptest.NewRecorder()
			handler.ServeHTTP(writer, req)
			resp := writer.Result()
			require.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}
}

func nextHandler() http2.HandlerFunc {
	return func(w http2.ResponseWriter, _ *http2.Request) {
		http.WriteResponse(w, http2.StatusOK, nil)
	}
}

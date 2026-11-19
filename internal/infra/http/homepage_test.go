package http_test

import (
	http2 "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bruli/raspberryWaterSystem/internal/infra/http"
)

func TestHomepage(t *testing.T) {
	tests := []struct {
		name, version string
	}{
		{name: "with a release version, then it returns ok with that version", version: "v2.1.1"},
		{name: "with a development build, then it returns ok with the dev version", version: "dev"},
	}
	for _, tt := range tests {
		t.Run(`Given a Homepage http handler,
		when a request is sent `+tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), http2.MethodGet, "/", http2.NoBody)
			writer := httptest.NewRecorder()
			http.Homepage(tt.version)(writer, req)
			resp := writer.Result()
			require.Equal(t, http2.StatusOK, resp.StatusCode)
			require.Equal(t, tt.version, resp.Header.Get(http.VersionHeader))
		})
	}
}

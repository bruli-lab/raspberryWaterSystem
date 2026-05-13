//go:build functional

package functional

import (
	"net/http"
	"testing"

	http2 "github.com/bruli/raspberryWaterSystem/internal/infra/http"

	"github.com/stretchr/testify/require"
)

func TestExecutionLogs(t *testing.T) {
	t.Run(`Given a execution logs endpoint,
	when a request is sent `, func(t *testing.T) {
		t.Run(`without authorization, then it returns an unauthorized`, func(t *testing.T) {
			resp, err := buildRequestAndSend(ctx, nil, nil, http.MethodGet, "/logs", cl)
			require.NoError(t, err)
			require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`with authorization, then it returns an valid response`, func(t *testing.T) {
			resp, err := buildRequestAndSend(ctx, nil, authorizationHeader(), http.MethodGet, "/logs", cl)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			var schema []http2.ExecutionLogsResponse
			readResponse(t, resp, &schema)
		})
	})
}

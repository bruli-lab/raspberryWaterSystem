//go:build functional

package functional

import (
	"net/http"
	"testing"

	http2 "github.com/bruli/raspberryWaterSystem/internal/infra/http"

	"github.com/stretchr/testify/require"
)

func TestStatus(t *testing.T) {
	t.Run(`Given a FindStatus endpoint,
	when a request is sent `, func(t *testing.T) {
		t.Run(`without authorization, then its return unauthorized`, func(t *testing.T) {
			resp, err := buildRequestAndSend(ctx, nil, nil, http.MethodGet, "/status", cl)
			require.NoError(t, err)
			require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`with authorization, then it return a valid response`, func(t *testing.T) {
			resp, err := buildRequestAndSend(ctx, nil, authorizationHeader(), http.MethodGet, "/status", cl)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			var schema http2.StatusResponse
			readResponse(t, resp, &schema)
			require.NotEmpty(t, schema.SystemStartedAt)
			require.True(t, schema.Active)
		})
	})
	t.Run(`Given an ActivateDeactivateServer endpoint,
	when a request is sent`, func(t *testing.T) {
		t.Run(`without authorization, then it returns unauthorized`, func(t *testing.T) {
			resp, err := buildRequestAndSend(ctx, nil, nil, http.MethodPatch, "/status/activate", cl)
			require.NoError(t, err)
			require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
		t.Run(`with authorization, then it returns a valid response`, func(t *testing.T) {
			resp, err := buildRequestAndSend(ctx, nil, authorizationHeader(), http.MethodPatch, "/status/deactivate", cl)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			respStatus, err := buildRequestAndSend(ctx, nil, authorizationHeader(), http.MethodGet, "/status", cl)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			var schema http2.StatusResponse
			readResponse(t, respStatus, &schema)
			require.False(t, schema.Active)

			_, err = buildRequestAndSend(ctx, nil, authorizationHeader(), http.MethodPatch, "/status/activate", cl)
			require.NoError(t, err)
		})
	})
}

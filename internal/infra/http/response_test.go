package http_test

import (
	"encoding/json"
	"io"
	http2 "net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func readResponse(t *testing.T, resp *http2.Response, schema any) {
	body := resp.Body
	defer func() { _ = body.Close() }()
	respBody, err := io.ReadAll(body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(respBody, &schema))
}

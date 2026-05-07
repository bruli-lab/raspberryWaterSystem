package http_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func buildRequestBody(body string) *strings.Reader {
	return strings.NewReader(body)
}

func buildRequestJsonToString(t *testing.T, req any) string {
	d, err := json.Marshal(req)
	require.NoError(t, err)
	return string(d)
}

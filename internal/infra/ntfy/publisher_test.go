//go:build infra

package ntfy_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/infra/ntfy"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/otel/trace/noop"
)

const ntfyImage = "binwiederhier/ntfy:v2.22.0"

type ntfyMessage struct {
	Event   string `json:"event"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

func startNtfy(t *testing.T) string {
	t.Helper()
	container, err := testcontainers.Run(
		t.Context(),
		ntfyImage,
		testcontainers.WithCmd("serve"),
		testcontainers.WithExposedPorts("80/tcp"),
		testcontainers.WithEnv(map[string]string{"NTFY_AUTH_DEFAULT_ACCESS": "read-write"}),
		testcontainers.WithWaitStrategy(wait.ForHTTP("/v1/health").WithPort("80/tcp")),
	)
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err)
	endpoint, err := container.PortEndpoint(t.Context(), "80/tcp", "http")
	require.NoError(t, err)
	return endpoint
}

func pollMessages(ctx context.Context, t *testing.T, serverURL, topic string) []ntfyMessage {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s/json?poll=1", serverURL, topic), http.NoBody)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var messages []ntfyMessage
	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var msg ntfyMessage
		require.NoError(t, decoder.Decode(&msg))
		messages = append(messages, msg)
	}
	return messages
}

func TestPublisher_Publish(t *testing.T) {
	serverURL := startNtfy(t)
	tests := []struct {
		name, topic, message string
	}{
		{
			name:    "with a plain message, then it returns nil error and the message is delivered to the topic",
			topic:   "bonsais",
			message: "testing",
		},
		{
			name:    "with a message on another topic, then it returns nil error and the message is delivered only to that topic",
			topic:   "garden",
			message: "zone executed during 20s",
		},
	}
	for _, tt := range tests {
		t.Run(`Given a ntfy Publisher,
		when Publish method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			pub, err := ntfy.NewPublisher("user", "pass", serverURL, tt.topic, noop.NewTracerProvider().Tracer("test"))
			require.NoError(t, err)

			err = pub.Publish(t.Context(), tt.message)
			require.NoError(t, err)

			messages := pollMessages(t.Context(), t, serverURL, tt.topic)
			require.Len(t, messages, 1)
			require.Equal(t, "message", messages[0].Event)
			require.Equal(t, "WaterSystem", messages[0].Title)
			require.Equal(t, tt.message, messages[0].Message)
		})
	}

	t.Run(`Given a ntfy Publisher pointing to an unreachable server,
	when Publish method is called,
	then it returns an error`, func(t *testing.T) {
		t.Parallel()
		pub, err := ntfy.NewPublisher("user", "pass", "http://127.0.0.1:1", "bonsais", noop.NewTracerProvider().Tracer("test"))
		require.NoError(t, err)

		err = pub.Publish(t.Context(), "testing")
		require.Error(t, err)
	})
}

//go:build infra

package ntfy_test

import (
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/infra/ntfy"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestPublisher_Publish(t *testing.T) {
	t.Run(`Given a ntfy Publisher,
	when Publish method is called,
	then it returns nil error`, func(t *testing.T) {
		pub, err := ntfy.NewPublisher("user", "pass", "http://localhost:8085", "bonsais", noop.NewTracerProvider().Tracer("test"))
		require.NoError(t, err)
		err = pub.Publish(t.Context(), "testing")
		require.NoError(t, err)
	})
}

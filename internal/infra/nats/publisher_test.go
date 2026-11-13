//go:build infra

package nats_test

import (
	"net"
	"net/netip"
	"strconv"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/infra/nats"
	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"github.com/google/uuid"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	natsgo "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	natsImage = "nats:2-alpine"
	subject   = "water.execution_log"
)

func startNats(t *testing.T) string {
	t.Helper()
	natsContainer, err := testcontainers.Run(
		t.Context(),
		natsImage,
		testcontainers.WithCmd("-js"),
		testcontainers.WithExposedPorts("4222/tcp"),
		testcontainers.WithWaitStrategy(wait.ForLog("Server is ready")),
	)
	testcontainers.CleanupContainer(t, natsContainer)
	require.NoError(t, err)
	endpoint, err := natsContainer.PortEndpoint(t.Context(), "4222/tcp", "nats")
	require.NoError(t, err)
	return endpoint
}

func TestPublisherWithoutServer(t *testing.T) {
	t.Run(`Given a nats Publisher built when the server is unreachable,
	when PublishEvent method is called,
	then it returns a not connected error without waiting for the publish timeout`, func(t *testing.T) {
		t.Parallel()
		pub, err := nats.NewPublisher("nats://127.0.0.1:1", []string{subject}, noop.NewTracerProvider().Tracer("test"))
		require.NoError(t, err)
		defer pub.Close()
		require.False(t, pub.IsConnected())

		started := time.Now()
		err = pub.PublishEvent(t.Context(), uuid.NewString(), subject, []byte(`{}`))

		require.ErrorAs(t, err, &nats.NotConnectedError{})
		require.Less(t, time.Since(started), time.Second)
	})
}

func TestPublisherPublishEvent(t *testing.T) {
	serverURL := startNats(t)
	t.Run(`Given a nats Publisher connected to a server without the stream,
	when PublishEvent method is called,
	then it creates the stream and publishes the event`, func(t *testing.T) {
		t.Parallel()
		pub, err := nats.NewPublisher(serverURL, []string{subject}, noop.NewTracerProvider().Tracer("test"))
		require.NoError(t, err)
		defer pub.Close()
		require.Eventually(t, pub.IsConnected, 5*time.Second, 50*time.Millisecond)

		require.NoError(t, pub.PublishEvent(t.Context(), uuid.NewString(), subject, []byte(`{"zone":"a"}`)))
		require.NoError(t, pub.PublishEvent(t.Context(), uuid.NewString(), subject, []byte(`{"zone":"b"}`)))

		nc, err := natsgo.Connect(serverURL)
		require.NoError(t, err)
		defer nc.Close()
		js, err := nc.JetStream()
		require.NoError(t, err)
		info, err := js.StreamInfo(nats.StreamName)
		require.NoError(t, err)
		require.Equal(t, []string{subject}, info.Config.Subjects)
		require.Equal(t, uint64(2), info.State.Msgs)
	})
}

func TestPublisherPublishEvents(t *testing.T) {
	serverURL := startNats(t)
	tests := []struct {
		name   string
		events int
	}{
		{name: "with a batch of events, then it publishes all of them without an error per event", events: 5},
	}
	for _, tt := range tests {
		t.Run(`Given a nats Publisher connected to a server without the stream,
		when PublishEvents method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			pub, err := nats.NewPublisher(serverURL, []string{subject}, noop.NewTracerProvider().Tracer("test"))
			require.NoError(t, err)
			defer pub.Close()
			require.Eventually(t, pub.IsConnected, 5*time.Second, 50*time.Millisecond)
			events := make([]outbox.Event, tt.events)
			for i := range events {
				events[i] = outbox.Event{ID: uuid.NewString(), EventType: subject, Payload: []byte(`{"zone":"a"}`)}
			}

			errs := pub.PublishEvents(t.Context(), events)

			require.Len(t, errs, tt.events)
			for _, err := range errs {
				require.NoError(t, err)
			}
			nc, err := natsgo.Connect(serverURL)
			require.NoError(t, err)
			defer nc.Close()
			js, err := nc.JetStream()
			require.NoError(t, err)
			info, err := js.StreamInfo(nats.StreamName)
			require.NoError(t, err)
			require.Equal(t, uint64(tt.events), info.State.Msgs)
		})
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)
	require.NoError(t, listener.Close())
	return addr.Port
}

func TestPublisherReconnecting(t *testing.T) {
	t.Run(`Given a nats Publisher built before the server is running,
	when the server starts and PublishEvent method is called,
	then it connects and publishes the event`, func(t *testing.T) {
		t.Parallel()
		port := freePort(t)
		pub, err := nats.NewPublisher("nats://127.0.0.1:"+strconv.Itoa(port), []string{subject}, noop.NewTracerProvider().Tracer("test"))
		require.NoError(t, err)
		defer pub.Close()
		require.ErrorAs(t, pub.PublishEvent(t.Context(), uuid.NewString(), subject, []byte(`{}`)), &nats.NotConnectedError{})

		natsContainer, err := testcontainers.Run(
			t.Context(),
			natsImage,
			testcontainers.WithCmd("-js"),
			testcontainers.WithExposedPorts("4222/tcp"),
			testcontainers.WithHostConfigModifier(func(hc *container.HostConfig) {
				hc.PortBindings = network.PortMap{
					network.MustParsePort("4222/tcp"): {{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: strconv.Itoa(port)}},
				}
			}),
			testcontainers.WithWaitStrategy(wait.ForLog("Server is ready")),
		)
		testcontainers.CleanupContainer(t, natsContainer)
		require.NoError(t, err)

		require.Eventually(t, pub.IsConnected, 10*time.Second, 100*time.Millisecond)
		require.NoError(t, pub.PublishEvent(t.Context(), uuid.NewString(), subject, []byte(`{"zone":"a"}`)))
	})
}

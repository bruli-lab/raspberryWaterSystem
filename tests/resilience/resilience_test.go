//go:build resilience

package resilience_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	infrahttp "github.com/bruli/raspberryWaterSystem/internal/infra/http"
	infranats "github.com/bruli/raspberryWaterSystem/internal/infra/nats"
	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	natsImage    = "nats:2-alpine"
	ntfyImage    = "binwiederhier/ntfy:v2.22.0"
	authToken    = "token"
	ntfyTopic    = "bonsais"
	outboxPeriod = 2 * time.Minute
)

type stack struct {
	nats, ntfy        testcontainers.Container
	natsURL, ntfyURL  string
	appURL, eventsDir string
	notificationsDir  string
	httpClient        *http.Client
}

func freePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		_ = listener.Close()
	}()
	addr, err := netip.ParseAddrPort(listener.Addr().String())
	require.NoError(t, err)
	return strconv.Itoa(int(addr.Port()))
}

func startContainerOnFixedPort(t *testing.T, image, containerPort, hostPort string, opts ...testcontainers.ContainerCustomizer) testcontainers.Container {
	t.Helper()
	port := network.MustParsePort(containerPort)
	opts = append(opts,
		testcontainers.WithExposedPorts(containerPort),
		testcontainers.WithHostConfigModifier(func(hostConfig *container.HostConfig) {
			hostConfig.PortBindings = network.PortMap{
				port: {{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: hostPort}},
			}
		}),
	)
	ctr, err := testcontainers.Run(t.Context(), image, opts...)
	testcontainers.CleanupContainer(t, ctr)
	require.NoError(t, err)
	return ctr
}

func buildApp(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "server")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "../../cmd/server")
	output, err := build.CombinedOutput()
	require.NoError(t, err, string(output))
	return binary
}

func startStack(t *testing.T) *stack {
	t.Helper()
	natsPort, ntfyPort, appPort := freePort(t), freePort(t), freePort(t)
	natsContainer := startContainerOnFixedPort(t, natsImage, "4222/tcp", natsPort,
		testcontainers.WithCmd("-js"),
		testcontainers.WithWaitStrategy(wait.ForLog("Server is ready")),
	)
	ntfyContainer := startContainerOnFixedPort(t, ntfyImage, "80/tcp", ntfyPort,
		testcontainers.WithCmd("serve"),
		testcontainers.WithEnv(map[string]string{"NTFY_AUTH_DEFAULT_ACCESS": "read-write"}),
		testcontainers.WithWaitStrategy(wait.ForHTTP("/v1/health").WithPort("80/tcp")),
	)
	dir := t.TempDir()
	st := &stack{
		nats:             natsContainer,
		ntfy:             ntfyContainer,
		natsURL:          "nats://127.0.0.1:" + natsPort,
		ntfyURL:          "http://127.0.0.1:" + ntfyPort,
		appURL:           "http://127.0.0.1:" + appPort,
		eventsDir:        filepath.Join(dir, "events"),
		notificationsDir: filepath.Join(dir, "events", "notifications"),
		httpClient:       &http.Client{Timeout: 5 * time.Second},
	}
	binary := buildApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	app := exec.CommandContext(ctx, binary)
	app.Env = append(os.Environ(),
		"SERVER_URL=127.0.0.1:"+appPort,
		"ENVIRONMENT=development",
		"ZONES_FILE="+filepath.Join(dir, "zones.yml"),
		"AUTH_TOKEN="+authToken,
		"DAILY_PROGRAMS_FILE="+filepath.Join(dir, "daily_programs.yml"),
		"ODD_PROGRAMS_FILE="+filepath.Join(dir, "odd_programs.yml"),
		"EVEN_PROGRAMS_FILE="+filepath.Join(dir, "even_programs.yml"),
		"WEEKLY_PROGRAMS_FILE="+filepath.Join(dir, "weekly_programs.yml"),
		"TEMPERATURE_PROGRAMS_FILE="+filepath.Join(dir, "temperature_programs.yml"),
		"EXECUTION_LOGS_FILE="+filepath.Join(dir, "execution_logs.json"),
		"STATUS_FILE="+filepath.Join(dir, "status.json"),
		"PENDING_IRRIGATIONS_DIRECTORY="+filepath.Join(dir, "pending_irrigations"),
		"NATS_SERVER_URL="+st.natsURL,
		"EVENTS_DIRECTORY="+st.eventsDir,
		"LOG_LEVEL=debug",
		"NTFY_URL="+st.ntfyURL,
		"NTFY_TOPIC="+ntfyTopic,
		"NTFY_USER=user",
		"NTFY_PASSWORD=password",
		"OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:"+freePort(t),
		"OTEL_TRACES_SAMPLER=always_on",
		"OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf",
	)
	var logs bytes.Buffer
	app.Stdout = &logs
	app.Stderr = &logs
	st.stopDependencies(t)
	require.NoError(t, app.Start())
	t.Cleanup(func() {
		cancel()
		_ = app.Wait()
		if t.Failed() {
			t.Logf("app logs:\n%s", logs.String())
		}
	})
	require.Eventually(t, func() bool {
		resp, err := st.request(t, http.MethodGet, "/", nil)
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 30*time.Second, 200*time.Millisecond, "app did not start without nats and ntfy")
	return st
}

func (s *stack) stopDependencies(t *testing.T) {
	t.Helper()
	timeout := 5 * time.Second
	require.NoError(t, s.nats.Stop(t.Context(), &timeout))
	require.NoError(t, s.ntfy.Stop(t.Context(), &timeout))
}

func (s *stack) startDependencies(t *testing.T) {
	t.Helper()
	require.NoError(t, s.nats.Start(t.Context()))
	require.NoError(t, s.ntfy.Start(t.Context()))
}

func (s *stack) request(t *testing.T, method, path string, body any) (*http.Response, error) {
	t.Helper()
	var buff bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buff).Encode(body))
	}
	req, err := http.NewRequestWithContext(t.Context(), method, s.appURL+path, &buff)
	require.NoError(t, err)
	req.Header.Set(infrahttp.AuthorizationHeader, authToken)
	return s.httpClient.Do(req)
}

func (s *stack) send(t *testing.T, method, path string, body any, expectedStatus int) {
	t.Helper()
	resp, err := s.request(t, method, path, body)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()
	require.Equal(t, expectedStatus, resp.StatusCode)
}

func (s *stack) executionLogs(t *testing.T) []infrahttp.ExecutionLogsResponse {
	t.Helper()
	resp, err := s.request(t, http.MethodGet, "/logs", nil)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var logs []infrahttp.ExecutionLogsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&logs))
	return logs
}

func countFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 0
	}
	require.NoError(t, err)
	files := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			files++
		}
	}
	return files
}

func (s *stack) lastPublishedExecutionLog(t *testing.T) outbox.ExecutionLog {
	t.Helper()
	nc, err := nats.Connect(s.natsURL)
	require.NoError(t, err)
	defer nc.Close()
	js, err := nc.JetStream()
	require.NoError(t, err)
	msg, err := js.GetLastMsg(infranats.StreamName, outbox.ExecutionLogsEventType)
	require.NoError(t, err)
	var executionLog outbox.ExecutionLog
	require.NoError(t, json.Unmarshal(msg.Data, &executionLog))
	return executionLog
}

func (s *stack) ntfyMessages(t *testing.T) []string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, fmt.Sprintf("%s/%s/json?poll=1", s.ntfyURL, ntfyTopic), http.NoBody)
	require.NoError(t, err)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	var messages []string
	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var msg struct {
			Message string `json:"message"`
		}
		if decoder.Decode(&msg) != nil {
			return messages
		}
		messages = append(messages, msg.Message)
	}
	return messages
}

func TestIrrigationWithoutNatsAndNtfy(t *testing.T) {
	tests := []struct {
		name, zoneID, zoneName string
		seconds                int
		expectedNotification   string
	}{
		{
			name:                 "then the zone is watered, the execution log is saved and the outbox is delivered when nats and ntfy come back",
			zoneID:               "bbf",
			zoneName:             "bonsai big",
			seconds:              1,
			expectedNotification: "bonsai big zone executed during 1s",
		},
	}
	for _, tt := range tests {
		t.Run(`Given the app running while nats and ntfy are down,
		when a zone is executed `+tt.name, func(t *testing.T) {
			t.Parallel()
			st := startStack(t)
			st.send(t, http.MethodPost, "/zones", infrahttp.Zone{Id: tt.zoneID, Name: tt.zoneName, Relays: []infrahttp.ZoneRelays{1, 2}}, http.StatusOK)

			st.send(t, http.MethodPost, "/zones/"+tt.zoneID+"/execute", infrahttp.ExecuteZoneRequest{Seconds: tt.seconds}, http.StatusOK)

			require.Eventually(t, func() bool {
				logs := st.executionLogs(t)
				return len(logs) == 1 && logs[0].ZoneName == tt.zoneName && logs[0].Seconds == tt.seconds
			}, 15*time.Second, 200*time.Millisecond, "execution log not saved while offline")
			require.Eventually(t, func() bool {
				return countFiles(t, st.eventsDir) == 1 && countFiles(t, st.notificationsDir) == 1
			}, 5*time.Second, 100*time.Millisecond, "outbox did not keep the event and the notification while offline")

			st.startDependencies(t)

			require.Eventually(t, func() bool {
				return countFiles(t, st.eventsDir) == 0 && countFiles(t, st.notificationsDir) == 0
			}, outboxPeriod, time.Second, "outbox not emptied after nats and ntfy came back")
			published := st.lastPublishedExecutionLog(t)
			require.Equal(t, tt.zoneName, published.ZoneName)
			require.Equal(t, tt.seconds, published.Seconds)
			require.Contains(t, st.ntfyMessages(t), tt.expectedNotification)
		})
	}
}

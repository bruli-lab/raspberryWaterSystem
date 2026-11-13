//go:build functional || resilience

package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
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
	natsImage     = "nats:2-alpine"
	ntfyImage     = "binwiederhier/ntfy:v2.22.0"
	serverPackage = "github.com/bruli/raspberryWaterSystem/cmd/server"
	AuthToken     = "token"
	NtfyTopic     = "bonsais"
)

type Stack struct {
	nats, ntfy       testcontainers.Container
	natsURL, ntfyURL string
	appURL           string
	eventsDir        string
	notificationsDir string
	client           *http.Client
}

type syncBuffer struct {
	m   sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()
	return b.buf.String()
}

func BuildApp(ctx context.Context) (binary string, cleanup func(), err error) {
	dir, err := os.MkdirTemp("", "water-system-app")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() {
		_ = os.RemoveAll(dir)
	}
	binary = filepath.Join(dir, "server")
	output, err := exec.CommandContext(ctx, "go", "build", "-o", binary, serverPackage).CombinedOutput()
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed building app: %w: %s", err, output)
	}
	return binary, cleanup, nil
}

func Start(t *testing.T, binary string) *Stack {
	t.Helper()
	return start(t, binary, false)
}

func StartWithDependenciesDown(t *testing.T, binary string) *Stack {
	t.Helper()
	return start(t, binary, true)
}

func start(t *testing.T, binary string, dependenciesDown bool) *Stack {
	t.Helper()
	natsPort, ntfyPort, appPort := freePort(t), freePort(t), freePort(t)
	st := &Stack{
		nats: startContainerOnFixedPort(t, natsImage, "4222/tcp", natsPort,
			testcontainers.WithCmd("-js"),
			testcontainers.WithWaitStrategy(wait.ForLog("Server is ready")),
		),
		ntfy: startContainerOnFixedPort(t, ntfyImage, "80/tcp", ntfyPort,
			testcontainers.WithCmd("serve"),
			testcontainers.WithEnv(map[string]string{"NTFY_AUTH_DEFAULT_ACCESS": "read-write"}),
			testcontainers.WithWaitStrategy(wait.ForHTTP("/v1/health").WithPort("80/tcp")),
		),
		natsURL: "nats://127.0.0.1:" + natsPort,
		ntfyURL: "http://127.0.0.1:" + ntfyPort,
		appURL:  "http://127.0.0.1:" + appPort,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
	dir := t.TempDir()
	st.eventsDir = filepath.Join(dir, "events")
	st.notificationsDir = filepath.Join(st.eventsDir, "notifications")
	if dependenciesDown {
		st.StopDependencies(t)
	}
	st.runApp(t, binary, dir, appPort)
	return st
}

func (s *Stack) runApp(t *testing.T, binary, dir, appPort string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	app := exec.CommandContext(ctx, binary)
	app.Env = append(os.Environ(),
		"SERVER_URL=127.0.0.1:"+appPort,
		"ENVIRONMENT=development",
		"ZONES_FILE="+filepath.Join(dir, "zones.yml"),
		"AUTH_TOKEN="+AuthToken,
		"DAILY_PROGRAMS_FILE="+filepath.Join(dir, "daily_programs.yml"),
		"ODD_PROGRAMS_FILE="+filepath.Join(dir, "odd_programs.yml"),
		"EVEN_PROGRAMS_FILE="+filepath.Join(dir, "even_programs.yml"),
		"WEEKLY_PROGRAMS_FILE="+filepath.Join(dir, "weekly_programs.yml"),
		"TEMPERATURE_PROGRAMS_FILE="+filepath.Join(dir, "temperature_programs.yml"),
		"EXECUTION_LOGS_FILE="+filepath.Join(dir, "execution_logs.json"),
		"STATUS_FILE="+filepath.Join(dir, "status.json"),
		"PENDING_IRRIGATIONS_DIRECTORY="+filepath.Join(dir, "pending_irrigations"),
		"NATS_SERVER_URL="+s.natsURL,
		"EVENTS_DIRECTORY="+s.eventsDir,
		"LOG_LEVEL=debug",
		"NTFY_URL="+s.ntfyURL,
		"NTFY_TOPIC="+NtfyTopic,
		"NTFY_USER=user",
		"NTFY_PASSWORD=password",
		"OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:"+freePort(t),
		"OTEL_TRACES_SAMPLER=always_on",
		"OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf",
	)
	logs := &syncBuffer{}
	app.Stdout = logs
	app.Stderr = logs
	require.NoError(t, app.Start())
	t.Cleanup(func() {
		cancel()
		_ = app.Wait()
		if t.Failed() {
			t.Logf("app logs:\n%s", logs.String())
		}
	})
	require.Eventually(t, func() bool {
		resp, err := s.Request(t, http.MethodGet, "/", nil, false)
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 30*time.Second, 200*time.Millisecond, "app did not start")
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

func (s *Stack) StopDependencies(t *testing.T) {
	t.Helper()
	timeout := 5 * time.Second
	require.NoError(t, s.nats.Stop(t.Context(), &timeout))
	require.NoError(t, s.ntfy.Stop(t.Context(), &timeout))
}

func (s *Stack) StartDependencies(t *testing.T) {
	t.Helper()
	require.NoError(t, s.nats.Start(t.Context()))
	require.NoError(t, s.ntfy.Start(t.Context()))
}

func (s *Stack) Request(t *testing.T, method, path string, body any, authorized bool) (*http.Response, error) {
	t.Helper()
	var buff bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buff).Encode(body))
	}
	req, err := http.NewRequestWithContext(t.Context(), method, s.appURL+path, &buff)
	require.NoError(t, err)
	if authorized {
		req.Header.Set(infrahttp.AuthorizationHeader, AuthToken)
	}
	return s.client.Do(req)
}

func (s *Stack) Send(t *testing.T, method, path string, body any) int {
	t.Helper()
	resp, err := s.Request(t, method, path, body, true)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

func (s *Stack) Get(t *testing.T, path string, schema any) {
	t.Helper()
	resp, err := s.Request(t, http.MethodGet, path, nil, true)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(schema))
}

func (s *Stack) CreateZone(t *testing.T, zone infrahttp.Zone) {
	t.Helper()
	require.Equal(t, http.StatusOK, s.Send(t, http.MethodPost, "/zones", zone))
}

func (s *Stack) ExecutionLogs(t *testing.T) []infrahttp.ExecutionLogsResponse {
	t.Helper()
	var logs []infrahttp.ExecutionLogsResponse
	s.Get(t, "/logs", &logs)
	return logs
}

func (s *Stack) EventsInOutbox(t *testing.T) int {
	t.Helper()
	return countFiles(t, s.eventsDir)
}

func (s *Stack) NotificationsInOutbox(t *testing.T) int {
	t.Helper()
	return countFiles(t, s.notificationsDir)
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

func (s *Stack) LastPublishedExecutionLog(t *testing.T) outbox.ExecutionLog {
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

func (s *Stack) NtfyMessages(t *testing.T) []string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, fmt.Sprintf("%s/%s/json?poll=1", s.ntfyURL, NtfyTopic), http.NoBody)
	require.NoError(t, err)
	resp, err := s.client.Do(req)
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

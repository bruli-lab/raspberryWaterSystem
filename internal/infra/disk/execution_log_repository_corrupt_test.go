package disk_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestExecutionLogRepositoryFindAllWithUnreadableFile(t *testing.T) {
	tests := []struct {
		name                string
		fileContent         *string
		unreadable          bool
		expectedErr         bool
		expectedLogs        int
		expectedCorruptFile bool
	}{
		{
			name:         "without an execution logs file, then it returns an empty list",
			expectedLogs: 0,
		},
		{
			name:         "with an empty execution logs file, then it returns an empty list",
			fileContent:  new(""),
			expectedLogs: 0,
		},
		{
			name:                "with a corrupt execution logs file, then it moves the file aside and returns an empty list",
			fileContent:         new(`[{"seconds":10,"zone_na`),
			expectedLogs:        0,
			expectedCorruptFile: true,
		},
		{
			name:        "with an execution logs file that cannot be read, then it returns an error and keeps the file",
			fileContent: new(`[]`),
			unreadable:  true,
			expectedErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an ExecutionLogRepository,
		when FindAll method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "execution_logs.json")
			if tt.fileContent != nil {
				require.NoError(t, os.WriteFile(path, []byte(*tt.fileContent), 0o600))
			}
			if tt.unreadable {
				require.NoError(t, os.Chmod(path, 0o200))
			}
			repo := disk.NewExecutionLogRepository(path, noop.NewTracerProvider().Tracer("test"))

			logs, err := repo.FindAll(t.Context())
			if tt.expectedErr {
				require.Error(t, err)
				_, statErr := os.Stat(path)
				require.NoError(t, statErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, logs, tt.expectedLogs)
			require.Equal(t, tt.expectedCorruptFile, hasCorruptExecutionLog(t, dir))
		})
	}
}

func TestExecutionLogRepositorySaveAfterCorruptFile(t *testing.T) {
	t.Run(`Given an ExecutionLogRepository with a corrupt execution logs file,
	when FindAll and Save methods are called,
	then it keeps the corrupt content in a separate file and writes the new logs`, func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "execution_logs.json")
		corrupt := `[{"seconds":10,"zone_na`
		require.NoError(t, os.WriteFile(path, []byte(corrupt), 0o600))
		repo := disk.NewExecutionLogRepository(path, noop.NewTracerProvider().Tracer("test"))

		logs, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), append(logs, fixtures.ExecutionLogBuilder{ZoneName: new("zone 1")}.Build())))

		saved, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		require.Len(t, saved, 1)
		require.Equal(t, corrupt, readCorruptExecutionLog(t, dir))
	})
}

func hasCorruptExecutionLog(t *testing.T, dir string) bool {
	t.Helper()
	return readCorruptExecutionLog(t, dir) != ""
}

func readCorruptExecutionLog(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".corrupt-") {
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			require.NoError(t, err)
			return string(data)
		}
	}
	return ""
}

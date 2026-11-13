package disk_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

const existingZonesYaml = `zone-a:
    name: existing zone
    relays:
        - 1
`

func TestZoneRepositorySaveWritesAtomically(t *testing.T) {
	tests := []struct {
		name            string
		readOnlyDir     bool
		expectedErr     bool
		expectedContent string
	}{
		{
			name:            "on a writable directory, then it replaces the file with 0644 permissions and leaves no temporary files",
			expectedContent: "existing zone",
		},
		{
			name:            "on a directory where the temporary file cannot be created, then it returns an error and keeps the previous file intact",
			readOnlyDir:     true,
			expectedErr:     true,
			expectedContent: existingZonesYaml,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a ZoneRepository with an existing zones file,
		when Save method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "zones.yml")
			require.NoError(t, os.WriteFile(path, []byte(existingZonesYaml), 0o600))
			if tt.readOnlyDir {
				require.NoError(t, os.Chmod(dir, 0o500))
				t.Cleanup(func() {
					_ = os.Chmod(dir, 0o700)
				})
			}
			repo := disk.NewZoneRepository(path, noop.NewTracerProvider().Tracer("test"))

			err := repo.Save(t.Context(), fixtures.ZoneBuilder{}.Build())

			content, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			entries, readErr := os.ReadDir(dir)
			require.NoError(t, readErr)
			require.Len(t, entries, 1)
			if tt.expectedErr {
				require.Error(t, err)
				require.Equal(t, tt.expectedContent, string(content))
				return
			}
			require.NoError(t, err)
			require.Contains(t, string(content), tt.expectedContent)
			info, statErr := os.Stat(path)
			require.NoError(t, statErr)
			require.Equal(t, os.FileMode(0o644), info.Mode().Perm())
		})
	}
}

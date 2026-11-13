package disk_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/bruli/raspberryWaterSystem/internal/infra/memory"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestPersistentStatusRepositorySave(t *testing.T) {
	tests := []struct {
		name           string
		fileContent    *string
		expectedActive bool
	}{
		{
			name:           "without a persisted status file, then it keeps the status active",
			expectedActive: true,
		},
		{
			name:           "with a persisted inactive status, then it restores the status as inactive",
			fileContent:    new(`{"active":false}`),
			expectedActive: false,
		},
		{
			name:           "with a persisted active status, then it keeps the status active",
			fileContent:    new(`{"active":true}`),
			expectedActive: true,
		},
		{
			name:           "with a corrupt persisted status file, then it keeps the status active",
			fileContent:    new(`{"act`),
			expectedActive: true,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a PersistentStatusRepository,
		when Save method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "status.json")
			if tt.fileContent != nil {
				require.NoError(t, os.WriteFile(path, []byte(*tt.fileContent), 0o600))
			}
			tracer := noop.NewTracerProvider().Tracer("test")
			repo := disk.NewPersistentStatusRepository(memory.NewStatusRepository(tracer), path, tracer)

			require.NoError(t, repo.Save(t.Context(), fixtures.StatusBuilder{Active: true}.Build()))

			st, err := repo.Find(t.Context())
			require.NoError(t, err)
			require.Equal(t, tt.expectedActive, st.IsActive())
		})
	}
}

func TestPersistentStatusRepositoryUpdate(t *testing.T) {
	tests := []struct {
		name           string
		readOnlyDir    bool
		expectedErr    bool
		expectedActive bool
	}{
		{
			name:           "deactivating the status, then it survives a restart as inactive",
			expectedActive: false,
		},
		{
			name:           "deactivating the status and the file cannot be written, then it returns an error and keeps the status active",
			readOnlyDir:    true,
			expectedErr:    true,
			expectedActive: true,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a PersistentStatusRepository with an active status,
		when Update method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "status.json")
			tracer := noop.NewTracerProvider().Tracer("test")
			repo := disk.NewPersistentStatusRepository(memory.NewStatusRepository(tracer), path, tracer)
			require.NoError(t, repo.Save(t.Context(), fixtures.StatusBuilder{Active: true}.Build()))
			if tt.readOnlyDir {
				require.NoError(t, os.Chmod(dir, 0o500))
				t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
			}

			st, err := repo.Find(t.Context())
			require.NoError(t, err)
			st.Deactivate()
			err = repo.Update(t.Context(), st)
			if tt.expectedErr {
				require.Error(t, err)
				current, err := repo.Find(t.Context())
				require.NoError(t, err)
				require.Equal(t, tt.expectedActive, current.IsActive())
				return
			}
			require.NoError(t, err)

			restarted := disk.NewPersistentStatusRepository(memory.NewStatusRepository(tracer), path, tracer)
			require.NoError(t, restarted.Save(t.Context(), fixtures.StatusBuilder{Active: true}.Build()))
			current, err := restarted.Find(t.Context())
			require.NoError(t, err)
			require.Equal(t, tt.expectedActive, current.IsActive())
		})
	}
}

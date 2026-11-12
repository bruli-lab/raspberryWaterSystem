package disk_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

const legacyProgramsYaml = `"10:00":
    - seconds: 20
      zones:
        - zone-a
        - zone-b
    - seconds: 30
      zones:
        - zone-c
`

const mixedProgramsYaml = `"10:00":
    - seconds: 20
      zones:
        - zone-a
"21:30":
    fertilizer: true
    executions:
        - seconds: 40
          zones:
            - zone-d
`

func TestFertilizerProgramRepositoryReadingLegacyFormat(t *testing.T) {
	tests := []struct {
		name, content, hour        string
		expectedFertilizer         bool
		expectedExecutions, length int
	}{
		{
			name:               "with a file in the legacy format, then it returns the program without fertilizer",
			content:            legacyProgramsYaml,
			hour:               "10:00",
			expectedExecutions: 2,
			length:             1,
		},
		{
			name:               "with a file mixing legacy and current formats, then it returns the legacy program without fertilizer",
			content:            mixedProgramsYaml,
			hour:               "10:00",
			expectedExecutions: 1,
			length:             2,
		},
		{
			name:               "with a file mixing legacy and current formats, then it returns the current program with fertilizer",
			content:            mixedProgramsYaml,
			hour:               "21:30",
			expectedFertilizer: true,
			expectedExecutions: 1,
			length:             2,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a FertilizerProgramRepository,
		when FindByHour and FindAll methods are called `+tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "programs.yml")
			require.NoError(t, os.WriteFile(path, []byte(tt.content), 0o600))
			repo := disk.NewFertilizerProgramRepository(path, noop.NewTracerProvider().Tracer("test"))
			hour, err := program.ParseHour(tt.hour)
			require.NoError(t, err)

			prg, err := repo.FindByHour(t.Context(), &hour)
			require.NoError(t, err)
			require.Equal(t, tt.expectedFertilizer, prg.IsFertilizer())
			require.Len(t, prg.Executions(), tt.expectedExecutions)

			all, err := repo.FindAll(t.Context())
			require.NoError(t, err)
			require.Len(t, all, tt.length)
		})
	}
}

func TestFertilizerProgramRepositoryMigratingLegacyFormat(t *testing.T) {
	t.Run(`Given a FertilizerProgramRepository with a file in the legacy format,
	when Save method is called with a new program,
	then it keeps the legacy programs and rewrites the file in the current format`, func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "programs.yml")
		require.NoError(t, os.WriteFile(path, []byte(legacyProgramsYaml), 0o600))
		repo := disk.NewFertilizerProgramRepository(path, noop.NewTracerProvider().Tracer("test"))
		hour, err := program.ParseHour("18:00")
		require.NoError(t, err)
		sec, err := program.ParseSeconds(15)
		require.NoError(t, err)
		exec, err := program.NewExecution(sec, []string{"zone-e"})
		require.NoError(t, err)
		prg, err := program.NewFertilizerProgram(hour, true, []program.Execution{exec})
		require.NoError(t, err)

		require.NoError(t, repo.Save(t.Context(), prg))

		all, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		require.Len(t, all, 2)
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Contains(t, string(content), "executions:")
		require.Contains(t, string(content), "fertilizer: false")
	})
}

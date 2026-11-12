package disk_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

const zonesWithInvalidRelayYaml = `zone-a:
    name: zone a
    relays:
        - 1
zone-b:
    name: zone b
    relays:
        - 99
`

const programsWithInvalidHourYaml = `"10:00":
    fertilizer: false
    executions:
        - seconds: 20
          zones:
            - zone-a
"25:99":
    fertilizer: false
    executions:
        - seconds: 20
          zones:
            - zone-b
`

const weeklyWithInvalidDayYaml = `Monday:
    "10:00":
        - seconds: 20
          zones:
            - zone-a
Someday:
    "10:00":
        - seconds: 20
          zones:
            - zone-b
`

func writeStoredFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestZoneRepositoryWithInvalidStoredRelay(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	t.Run(`Given a ZoneRepository with a zone stored with an unknown relay,
	when FindAll method is called,
	then it skips the invalid zone and returns the valid ones`, func(t *testing.T) {
		t.Parallel()
		repo := disk.NewZoneRepository(writeStoredFile(t, zonesWithInvalidRelayYaml), tracer)
		zones, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		require.Len(t, zones, 1)
		require.Equal(t, "zone-a", zones[0].Id())
	})
	t.Run(`Given a ZoneRepository with a zone stored with an unknown relay,
	when FindByID method is called with that zone,
	then it returns an invalid stored entry error`, func(t *testing.T) {
		t.Parallel()
		repo := disk.NewZoneRepository(writeStoredFile(t, zonesWithInvalidRelayYaml), tracer)
		_, err := repo.FindByID(t.Context(), "zone-b")
		require.ErrorAs(t, err, &disk.InvalidStoredEntryError{})
	})
}

func TestProgramRepositoriesWithInvalidStoredEntries(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	t.Run(`Given a FertilizerProgramRepository with a program stored with an invalid hour,
	when FindAll method is called,
	then it skips the invalid program and returns the valid ones`, func(t *testing.T) {
		t.Parallel()
		repo := disk.NewFertilizerProgramRepository(writeStoredFile(t, programsWithInvalidHourYaml), tracer)
		programs, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		require.Len(t, programs, 1)
		require.Equal(t, "10:00", programs[0].Hour().String())
	})
	t.Run(`Given a WeeklyRepository with a program stored with an invalid week day,
	when FindAll method is called,
	then it skips the invalid day and returns the valid ones`, func(t *testing.T) {
		t.Parallel()
		repo := disk.NewWeeklyRepository(writeStoredFile(t, weeklyWithInvalidDayYaml), tracer)
		weeklies, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		require.Len(t, weeklies, 1)
		require.Equal(t, "Monday", weeklies[0].WeekDay().String())
	})
}

package disk_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
	"github.com/bruli/raspberryWaterSystem/internal/fixtures"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

const programsWithStatusYaml = `"10:00":
    - seconds: 20
      zones:
        - zone-a
"11:00":
    executions:
        - seconds: 20
          zones:
            - zone-b
"12:00":
    status: disabled
    executions:
        - seconds: 20
          zones:
            - zone-c
"13:00":
    status: paused
    executions:
        - seconds: 20
          zones:
            - zone-d
`

func TestProgramRepositoryReadingStatus(t *testing.T) {
	tests := []struct {
		name, hour     string
		expectedErr    any
		expectedStatus program.Status
	}{
		{
			name:           "with a program in the legacy format, then it returns it enabled",
			hour:           "10:00",
			expectedStatus: program.EnabledStatus,
		},
		{
			name:           "with a program without status, then it returns it enabled",
			hour:           "11:00",
			expectedStatus: program.EnabledStatus,
		},
		{
			name:           "with a disabled program, then it returns it disabled",
			hour:           "12:00",
			expectedStatus: program.DisabledStatus,
		},
		{
			name:        "with an unknown status, then it returns an invalid stored entry error",
			hour:        "13:00",
			expectedErr: &disk.InvalidStoredEntryError{},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a ProgramRepository, when FindByHour method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "programs.yml")
			require.NoError(t, os.WriteFile(path, []byte(programsWithStatusYaml), 0o600))
			repo := disk.NewProgramRepository(path, noop.NewTracerProvider().Tracer("test"))
			hour, err := program.ParseHour(tt.hour)
			require.NoError(t, err)

			prg, err := repo.FindByHour(t.Context(), &hour)
			if tt.expectedErr != nil {
				require.ErrorAs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expectedStatus, prg.Status())
		})
	}
}

func TestProgramRepositoryFindAllSkipsInvalidStatus(t *testing.T) {
	t.Run(`Given a ProgramRepository with a program with an unknown status,
	when FindAll method is called,
	then it returns the rest of programs`, func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "programs.yml")
		require.NoError(t, os.WriteFile(path, []byte(programsWithStatusYaml), 0o600))
		repo := disk.NewProgramRepository(path, noop.NewTracerProvider().Tracer("test"))

		all, err := repo.FindAll(t.Context())

		require.NoError(t, err)
		require.Len(t, all, 3)
	})
}

func TestRepositoriesSavingStatus(t *testing.T) {
	hour, err := program.ParseHour("07:30")
	require.NoError(t, err)
	tests := []struct {
		name   string
		status program.Status
	}{
		{name: "with a disabled program, then it is read back disabled", status: program.DisabledStatus},
		{name: "with an enabled program, then it is read back enabled", status: program.EnabledStatus},
	}
	for _, tt := range tests {
		t.Run(`Given the program repositories, when Save method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			tracer := noop.NewTracerProvider().Tracer("test")
			dir := t.TempDir()
			prg := fixtures.ProgramBuilder{Hour: &hour, Status: &tt.status}.Build()

			daily := disk.NewProgramRepository(filepath.Join(dir, "daily.yml"), tracer)
			require.NoError(t, daily.Save(t.Context(), &prg))
			foundDaily, err := daily.FindByHour(t.Context(), &hour)
			require.NoError(t, err)
			require.Equal(t, tt.status, foundDaily.Status())

			day := program.WeekDay(time.Monday)
			weekly := fixtures.WeeklyBuilder{WeekDay: &day, Programs: []program.Program{prg}}.Build()
			weeklyRepo := disk.NewWeeklyRepository(filepath.Join(dir, "weekly.yml"), tracer)
			require.NoError(t, weeklyRepo.Save(t.Context(), &weekly))
			foundWeekly, err := weeklyRepo.FindByDayAndHour(t.Context(), &day, &hour)
			require.NoError(t, err)
			require.Len(t, foundWeekly.Programs(), 1)
			require.Equal(t, tt.status, foundWeekly.Programs()[0].Status())

			temperature := float32(25)
			temp := fixtures.TemperatureBuilder{Temperature: &temperature, Programs: []program.Program{prg}}.Build()
			tempRepo := disk.NewTemperatureProgramRepository(filepath.Join(dir, "temperature.yml"), tracer)
			require.NoError(t, tempRepo.Save(t.Context(), &temp))
			foundTemp, err := tempRepo.FindByTemperatureAndHour(t.Context(), temperature, hour)
			require.NoError(t, err)
			require.Len(t, foundTemp.Programs(), 1)
			require.Equal(t, tt.status, foundTemp.Programs()[0].Status())
		})
	}
}

const legacyWeeklyYaml = `Monday:
    "07:30":
        - seconds: 20
          zones:
            - zone-a
`

func TestWeeklyRepositoryReadingLegacyFormat(t *testing.T) {
	t.Run(`Given a WeeklyRepository with a file in the legacy format,
	when FindByDayAndHour method is called,
	then it returns the program enabled`, func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "weekly.yml")
		require.NoError(t, os.WriteFile(path, []byte(legacyWeeklyYaml), 0o600))
		repo := disk.NewWeeklyRepository(path, noop.NewTracerProvider().Tracer("test"))
		day := program.WeekDay(time.Monday)
		hour, err := program.ParseHour("07:30")
		require.NoError(t, err)

		found, err := repo.FindByDayAndHour(t.Context(), &day, &hour)

		require.NoError(t, err)
		require.Len(t, found.Programs(), 1)
		require.Equal(t, program.EnabledStatus, found.Programs()[0].Status())
		require.Len(t, found.Programs()[0].Executions(), 1)
	})
}

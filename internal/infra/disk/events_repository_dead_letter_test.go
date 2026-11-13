package disk_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestEventsRepositoryFindAllWithUnreadableFiles(t *testing.T) {
	tests := []struct {
		name                string
		files               map[string]string
		createDir           bool
		expectedEvents      int
		expectedDeadLetters []string
		expectedRemaining   []string
	}{
		{
			name:      "with a corrupt event file, then it returns the valid events and moves the corrupt file to the dead letter directory",
			createDir: true,
			files: map[string]string{
				"1.json": `{"id":"truncat`,
			},
			expectedEvents:      2,
			expectedDeadLetters: []string{"1.json"},
		},
		{
			name:      "with a leftover temporary file, then it ignores it",
			createDir: true,
			files: map[string]string{
				".2.json.123.tmp": `{"id":"partial`,
			},
			expectedEvents:    2,
			expectedRemaining: []string{".2.json.123.tmp"},
		},
		{
			name:           "and the events directory does not exist, then it returns no events without error",
			expectedEvents: 0,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an EventsRepository,
		when FindAll method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(t.TempDir(), "events")
			repo := disk.NewEventsRepository(dir, noop.NewTracerProvider().Tracer("test"))
			if tt.createDir {
				for i := range 2 {
					ev, err := outbox.NewFromWeather(t.Context(), outbox.Weather{Temperature: 20, Humidity: 50}, time.Now())
					require.NoError(t, err)
					ev.EventAt = ev.EventAt.Add(time.Duration(i) * time.Second)
					require.NoError(t, repo.Save(t.Context(), ev))
				}
				for name, content := range tt.files {
					require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
				}
			}

			events, err := repo.FindAll(t.Context())

			require.NoError(t, err)
			require.Len(t, events, tt.expectedEvents)
			for _, ev := range events {
				require.Equal(t, outbox.TerraceWeatherEventType, ev.EventType)
			}
			for _, name := range tt.expectedDeadLetters {
				require.FileExists(t, filepath.Join(dir, disk.DeadLetterDirectory, name))
				require.NoFileExists(t, filepath.Join(dir, name))
			}
			for _, name := range tt.expectedRemaining {
				require.FileExists(t, filepath.Join(dir, name))
			}
		})
	}
}

package disk_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func executedEvent(zoneName string) zone.Executed {
	return zone.Executed{
		BasicEvent: cqs.NewBasicEvent(zone.ExecutedEventName, uuid.New(), "zone-id"),
		ZoneID:     "zone-id",
		ZoneName:   zoneName,
		Seconds:    20,
		RelayPins:  []string{"18"},
	}
}

func fertilizerEvent() zone.FertilizerZoneExecuted {
	return zone.FertilizerZoneExecuted{
		BasicEvent:            cqs.NewBasicEvent(zone.FertilizerZoneExecutedEventName, uuid.New(), "zone-id"),
		ZoneID:                "zone-id",
		ZoneName:              "fertilizer zone",
		ZoneSeconds:           30,
		ZoneRelayPins:         []string{"18", "24"},
		FertilizerPumpSeconds: 5,
	}
}

func pendingIrrigationsRepository(dir string) disk.PendingIrrigationsRepository {
	return disk.NewPendingIrrigationsRepository(dir, time.Now, noop.NewTracerProvider().Tracer("test"))
}

func TestPendingIrrigationsRepositorySaveAndFindAll(t *testing.T) {
	tests := []struct {
		name   string
		events []cqs.Event
	}{
		{
			name:   "with a plain zone execution, then it restores the same event",
			events: []cqs.Event{executedEvent("zone 1")},
		},
		{
			name:   "with several executions of different kinds, then it restores all of them in the order they were saved",
			events: []cqs.Event{executedEvent("zone 1"), fertilizerEvent(), executedEvent("zone 2")},
		},
	}
	for _, tt := range tests {
		t.Run(`Given a PendingIrrigationsRepository,
		when Save and FindAll methods are called `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := pendingIrrigationsRepository(filepath.Join(t.TempDir(), "pending"))
			for _, ev := range tt.events {
				_, err := repo.Save(t.Context(), ev)
				require.NoError(t, err)
			}

			pendings, err := repo.FindAll(t.Context())
			require.NoError(t, err)
			require.Len(t, pendings, len(tt.events))
			for i, pending := range pendings {
				require.Equal(t, tt.events[i].EventID(), pending.Event.EventID())
				require.Equal(t, tt.events[i].EventName(), pending.Event.EventName())
				require.Equal(t, tt.events[i], withSameTime(tt.events[i], pending.Event))
			}
		})
	}
}

func withSameTime(expected, restored cqs.Event) cqs.Event {
	switch ev := restored.(type) {
	case zone.Executed:
		ev.AtAttr = expected.EventAt()
		return ev
	case zone.FertilizerZoneExecuted:
		ev.AtAttr = expected.EventAt()
		return ev
	default:
		return restored
	}
}

func TestPendingIrrigationsRepositorySaveUnsupportedEvent(t *testing.T) {
	t.Run(`Given a PendingIrrigationsRepository,
	when Save method is called with a clean cycle event,
	then it returns an unsupported pending irrigation error and persists nothing`, func(t *testing.T) {
		t.Parallel()
		repo := pendingIrrigationsRepository(filepath.Join(t.TempDir(), "pending"))
		_, err := repo.Save(t.Context(), zone.CleanZoneExecuted{
			BasicEvent: cqs.NewBasicEvent(zone.CleanZoneExecutedEventName, uuid.New(), "zone-id"),
		})
		require.ErrorAs(t, err, &disk.UnsupportedPendingIrrigationError{})

		pendings, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		require.Empty(t, pendings)
	})
}

func TestPendingIrrigationsRepositoryRemove(t *testing.T) {
	t.Run(`Given a PendingIrrigationsRepository with two pending irrigations,
	when Remove method is called with the first one,
	then only the second one remains`, func(t *testing.T) {
		t.Parallel()
		repo := pendingIrrigationsRepository(filepath.Join(t.TempDir(), "pending"))
		first, err := repo.Save(t.Context(), executedEvent("zone 1"))
		require.NoError(t, err)
		second, err := repo.Save(t.Context(), executedEvent("zone 2"))
		require.NoError(t, err)

		require.NoError(t, repo.Remove(t.Context(), first.ID))
		require.NoError(t, repo.Remove(t.Context(), first.ID))

		pendings, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		require.Len(t, pendings, 1)
		require.Equal(t, second.ID, pendings[0].ID)
	})
}

func TestPendingIrrigationsRepositoryFindAllWithCorruptFile(t *testing.T) {
	t.Run(`Given a PendingIrrigationsRepository with a corrupt pending irrigation file,
	when FindAll method is called,
	then it skips the corrupt file, moves it to the dead letter directory and returns the valid ones`, func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join(t.TempDir(), "pending")
		repo := pendingIrrigationsRepository(dir)
		saved, err := repo.Save(t.Context(), executedEvent("zone 1"))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "00000000000000000001-0000000001.json"), []byte(`{"name":`), 0o600))

		pendings, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		require.Len(t, pendings, 1)
		require.Equal(t, saved.ID, pendings[0].ID)
		_, err = os.Stat(filepath.Join(dir, disk.DeadLetterDirectory, "00000000000000000001-0000000001.json"))
		require.NoError(t, err)
	})
}

func TestPendingIrrigationsRepositoryFindAllWithoutDirectory(t *testing.T) {
	t.Run(`Given a PendingIrrigationsRepository without directory,
	when FindAll method is called,
	then it returns no pending irrigations and no error`, func(t *testing.T) {
		t.Parallel()
		repo := pendingIrrigationsRepository(filepath.Join(t.TempDir(), "missing"))
		pendings, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		require.Empty(t, pendings)
	})
}

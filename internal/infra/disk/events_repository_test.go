//go:build infra

package disk_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"github.com/stretchr/testify/require"
)

func eventsRepositoryWithExecutionLogs(t *testing.T, count int) *disk.EventsRepository {
	t.Helper()
	repo := disk.NewEventsRepository(tempFile(t, "events"), tracer())
	now := time.Date(2026, time.November, 13, 9, 0, 0, 0, time.UTC)
	for i := range count {
		lo := outbox.ExecutionLog{
			Seconds:    10,
			ZoneName:   fmt.Sprintf("zone_%v", i),
			ExecutedAt: now.Add(10 * time.Minute * time.Duration(i)),
		}
		ev, err := outbox.NewFromExecutionLog(t.Context(), lo, now.Add(time.Duration(i)*time.Microsecond))
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), ev))
	}
	return repo
}

func TestEventsRepositoryFindAll(t *testing.T) {
	t.Run(`Given an EventsRepository with three saved execution log events, when FindAll method is called, then it returns the three events with a readable payload`, func(t *testing.T) {
		t.Parallel()
		events, err := eventsRepositoryWithExecutionLogs(t, 3).FindAll(t.Context())
		require.NoError(t, err)
		require.Len(t, events, 3)
		for _, ev := range events {
			require.Equal(t, outbox.ExecutionLogsEventType, ev.EventType)
			var lo outbox.ExecutionLog
			require.NoError(t, json.Unmarshal(ev.Payload, &lo))
			require.Equal(t, 10, lo.Seconds)
		}
	})
}

func TestEventsRepositoryRemove(t *testing.T) {
	t.Run(`Given an EventsRepository with three saved events, when Remove method is called for each of them, then no event remains`, func(t *testing.T) {
		t.Parallel()
		repo := eventsRepositoryWithExecutionLogs(t, 3)
		events, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		for i := range events {
			require.NoError(t, repo.Remove(t.Context(), &events[i]))
		}
		remaining, err := repo.FindAll(t.Context())
		require.NoError(t, err)
		require.Empty(t, remaining)
	})
}

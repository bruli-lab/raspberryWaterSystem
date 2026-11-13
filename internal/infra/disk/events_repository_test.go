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

func TestEventsRepository(t *testing.T) {
	t.Run(`Given an Events repository`, func(t *testing.T) {
		repo := disk.NewEventsRepository("/tmp/events", tracer())
		number := 3
		var events []outbox.Event
		t.Run(`when Save method is called with a Log payload in event,
		then it create files`, func(t *testing.T) {
			for i := 0; i < number; i++ {
				now := time.Now()
				lo := outbox.ExecutionLog{
					Seconds:    10,
					ZoneName:   fmt.Sprintf("zone_%v", i),
					ExecutedAt: now.Add(time.Minute * time.Duration(i) * 10),
				}
				event, err := outbox.NewFromExecutionLog(t.Context(), lo, now.Add(time.Duration(i)*time.Microsecond))
				require.NoError(t, err)
				err = repo.Save(t.Context(), event)
				require.NoError(t, err)
			}
		})
		t.Run(`when FindAll method it's called,
		then it return a valid slice and log payload is unmarshall`, func(t *testing.T) {
			got, err := repo.FindAll(t.Context())
			require.NoError(t, err)
			require.Len(t, got, number)
			events = got

			for _, ev := range events {
				var log outbox.ExecutionLog
				err = json.Unmarshal(ev.Payload, &log)
				require.NoError(t, err)
			}
		})
		t.Run(`when Remove method it's called,
			then it delete the json file`, func(t *testing.T) {
			for i := range events {
				err := repo.Remove(t.Context(), &events[i])
				require.NoError(t, err)
			}
		})
	})
}

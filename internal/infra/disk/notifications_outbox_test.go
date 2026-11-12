package disk_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/infra/disk"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func notificationsOutbox(dir string) disk.NotificationsOutbox {
	return disk.NewNotificationsOutbox(dir, time.Now, noop.NewTracerProvider().Tracer("test"))
}

func TestNotificationsOutbox(t *testing.T) {
	tests := []struct {
		name     string
		messages []string
	}{
		{name: "with a single message, then it keeps the message until it is removed", messages: []string{"zone a executed during 20s"}},
		{name: "with several messages, then it returns them in the order they were published", messages: []string{"first", "second", "third"}},
	}
	for _, tt := range tests {
		t.Run(`Given a NotificationsOutbox,
		when Publish and FindAll methods are called `+tt.name, func(t *testing.T) {
			t.Parallel()
			outbox := notificationsOutbox(filepath.Join(t.TempDir(), "notifications"))
			for _, message := range tt.messages {
				require.NoError(t, outbox.Publish(t.Context(), message))
			}

			notifications, err := outbox.FindAll(t.Context())
			require.NoError(t, err)
			require.Len(t, notifications, len(tt.messages))
			for i, notification := range notifications {
				require.Equal(t, tt.messages[i], notification.Message)
				require.False(t, notification.CreatedAt.IsZero())
			}

			for _, notification := range notifications {
				require.NoError(t, outbox.Remove(t.Context(), notification.ID))
			}
			notifications, err = outbox.FindAll(t.Context())
			require.NoError(t, err)
			require.Empty(t, notifications)
		})
	}
}

func TestNotificationsOutboxWithCorruptFile(t *testing.T) {
	t.Run(`Given a NotificationsOutbox with a corrupt notification file,
	when FindAll method is called,
	then it moves the corrupt file to the dead letter directory and returns the valid notifications`, func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join(t.TempDir(), "notifications")
		outbox := notificationsOutbox(dir)
		require.NoError(t, outbox.Publish(t.Context(), "valid"))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "00000000000000000001-0000000001.json"), []byte(`{"mess`), 0o600))

		notifications, err := outbox.FindAll(t.Context())
		require.NoError(t, err)
		require.Len(t, notifications, 1)
		require.Equal(t, "valid", notifications[0].Message)
		_, err = os.Stat(filepath.Join(dir, disk.DeadLetterDirectory, "00000000000000000001-0000000001.json"))
		require.NoError(t, err)
	})
}

func TestNotificationsOutboxWithoutDirectory(t *testing.T) {
	t.Run(`Given a NotificationsOutbox without directory,
	when FindAll method is called,
	then it returns no notifications and no error`, func(t *testing.T) {
		t.Parallel()
		notifications, err := notificationsOutbox(filepath.Join(t.TempDir(), "missing")).FindAll(t.Context())
		require.NoError(t, err)
		require.Empty(t, notifications)
	})
}

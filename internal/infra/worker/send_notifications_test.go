package worker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"github.com/bruli/raspberryWaterSystem/internal/infra/worker"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

type pendingNotifications struct {
	notifications []outbox.Notification
	findErr       error
	removed       []string
}

func (p *pendingNotifications) FindAll(_ context.Context) ([]outbox.Notification, error) {
	return p.notifications, p.findErr
}

func (p *pendingNotifications) Remove(_ context.Context, id string) error {
	p.removed = append(p.removed, id)
	return nil
}

type notificationSender struct {
	failFrom int
	sent     []string
}

func (n *notificationSender) Publish(_ context.Context, message string) error {
	if n.failFrom > 0 && len(n.sent)+1 >= n.failFrom {
		return errors.New("ntfy unreachable")
	}
	n.sent = append(n.sent, message)
	return nil
}

func TestSendNotifications(t *testing.T) {
	location, err := time.LoadLocation("Europe/Madrid")
	require.NoError(t, err)
	now := time.Date(2026, time.November, 12, 18, 0, 0, 0, time.UTC)
	recent := outbox.Notification{ID: "1", Message: "zone a executed during 20s", CreatedAt: now.Add(-10 * time.Second)}
	delayed := outbox.Notification{ID: "2", Message: "zone b executed during 30s", CreatedAt: now.Add(-3 * time.Hour)}
	tests := []struct {
		name            string
		notifications   []outbox.Notification
		findErr         error
		failFrom        int
		expectedErr     bool
		expectedSent    []string
		expectedRemoved []string
	}{
		{
			name:            "with recent and delayed notifications, then it sends all of them in order, adds the original time to the delayed ones and removes them",
			notifications:   []outbox.Notification{recent, delayed},
			expectedSent:    []string{"zone a executed during 20s", "zone b executed during 30s (12/11 16:00)"},
			expectedRemoved: []string{"1", "2"},
		},
		{
			name:            "and the sender fails, then it stops at the first failure and keeps the unsent notifications",
			notifications:   []outbox.Notification{recent, delayed},
			failFrom:        2,
			expectedErr:     true,
			expectedSent:    []string{"zone a executed during 20s"},
			expectedRemoved: []string{"1"},
		},
		{
			name:        "and pending notifications cannot be read, then it returns an error without sending anything",
			findErr:     errors.New("io error"),
			expectedErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(`Given pending notifications,
		when SendNotifications is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			pending := &pendingNotifications{notifications: tt.notifications, findErr: tt.findErr}
			sender := &notificationSender{failFrom: tt.failFrom}

			sent, err := worker.SendNotifications(t.Context(), pending, sender, now, location, noop.NewTracerProvider().Tracer("test"))
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expectedSent, sender.sent)
			require.Equal(t, tt.expectedRemoved, pending.removed)
			require.Equal(t, len(tt.expectedSent), sent)
		})
	}
}

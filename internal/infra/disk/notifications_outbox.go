package disk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const NotificationsDirectory = "notifications"

type Notification struct {
	ID        string
	Message   string
	CreatedAt time.Time
}

type notificationFile struct {
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationsOutbox struct {
	dir      string
	tracer   trace.Tracer
	sequence *atomic.Uint64
	now      func() time.Time
}

func (n NotificationsOutbox) Publish(ctx context.Context, message string) error {
	_, span := n.tracer.Start(ctx, "NotificationsOutbox.Publish")
	defer span.End()
	if err := n.save(message); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "notification queued")
	return nil
}

func (n NotificationsOutbox) save(message string) error {
	createdAt := n.now()
	id := fmt.Sprintf("%020d-%010d", createdAt.UnixNano(), n.sequence.Add(1))
	if err := os.MkdirAll(n.dir, 0o755); err != nil {
		return fmt.Errorf("failed to create notifications directory: %w", err)
	}
	return writeJsonFile(n.path(id), notificationFile{Message: message, CreatedAt: createdAt})
}

func (n NotificationsOutbox) FindAll(ctx context.Context) ([]Notification, error) {
	_, span := n.tracer.Start(ctx, "NotificationsOutbox.FindAll")
	defer span.End()
	entries, err := os.ReadDir(n.dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			span.SetStatus(codes.Ok, "notifications directory does not exist yet")
			return nil, nil
		}
		err = fmt.Errorf("failed to read notifications directory: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	notifications := make([]Notification, 0, len(entries))
	for _, entry := range entries {
		if !isEventFile(entry) {
			continue
		}
		path := filepath.Join(n.dir, entry.Name())
		notification, err := readNotification(path)
		if err != nil {
			span.RecordError(err)
			if err := moveToDeadLetter(n.dir, path); err != nil {
				span.RecordError(err)
			}
			continue
		}
		notifications = append(notifications, notification)
	}
	slices.SortFunc(notifications, func(a, b Notification) int {
		return strings.Compare(a.ID, b.ID)
	})
	span.SetStatus(codes.Ok, "notifications found")
	return notifications, nil
}

func readNotification(path string) (Notification, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Notification{}, fmt.Errorf("failed to read notification %s: %w", path, err)
	}
	var file notificationFile
	if err = json.Unmarshal(data, &file); err != nil {
		return Notification{}, fmt.Errorf("failed to decode notification %s: %w", path, err)
	}
	id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return Notification{ID: id, Message: file.Message, CreatedAt: file.CreatedAt}, nil
}

func (n NotificationsOutbox) Remove(ctx context.Context, id string) error {
	_, span := n.tracer.Start(ctx, "NotificationsOutbox.Remove")
	defer span.End()
	if err := os.Remove(n.path(id)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		err = fmt.Errorf("failed to remove notification %s: %w", id, err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "notification removed")
	return nil
}

func (n NotificationsOutbox) path(id string) string {
	return filepath.Join(n.dir, id+".json")
}

func NewNotificationsOutbox(dir string, now func() time.Time, tracer trace.Tracer) NotificationsOutbox {
	return NotificationsOutbox{dir: dir, tracer: tracer, sequence: &atomic.Uint64{}, now: now}
}

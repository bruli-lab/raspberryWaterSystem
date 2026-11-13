package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/infra/outbox"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	notificationDelayThreshold = time.Minute
	delayedNotificationLayout  = "02/01 15:04"
)

type PendingNotifications interface {
	FindAll(ctx context.Context) ([]outbox.Notification, error)
	Remove(ctx context.Context, id string) error
}

type NotificationSender interface {
	Publish(ctx context.Context, message string) error
}

func SendNotifications(
	ctx context.Context,
	pending PendingNotifications,
	sender NotificationSender,
	now time.Time,
	location *time.Location,
	tracer trace.Tracer,
) (int, error) {
	ctx, span := tracer.Start(ctx, "SendNotifications worker")
	defer span.End()
	notifications, err := pending.FindAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}
	sent := 0
	for _, notification := range notifications {
		if err = sender.Publish(ctx, notificationMessage(notification, now, location)); err != nil {
			err = SendNotificationError{err: err}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return sent, err
		}
		sent++
		if err = pending.Remove(ctx, notification.ID); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return sent, err
		}
	}
	span.SetAttributes(attribute.Int("notifications.sent", sent))
	span.SetStatus(codes.Ok, "notifications sent")
	return sent, nil
}

func notificationMessage(notification outbox.Notification, now time.Time, location *time.Location) string {
	if now.Sub(notification.CreatedAt) < notificationDelayThreshold {
		return notification.Message
	}
	return fmt.Sprintf("%s (%s)", notification.Message, notification.CreatedAt.In(location).Format(delayedNotificationLayout))
}

type SendNotificationError struct {
	err error
}

func (s SendNotificationError) Error() string {
	return fmt.Sprintf("failed sending notification: %q", s.err.Error())
}

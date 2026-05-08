package telegram

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type MessagePublisher struct {
	chatID int
	tracer trace.Tracer
	bot    *tgbotapi.BotAPI
}

func NewMessagePublisher(token string, chatID int, tracer trace.Tracer, isProd bool) (*MessagePublisher, error) {
	var (
		bot *tgbotapi.BotAPI
		err error
	)
	switch {
	case isProd:
		bot, err = tgbotapi.NewBotAPI(token)
	default:
		bot, err = tgbotapi.NewBotAPIWithAPIEndpoint(token, "http://mockserver:1080/bot%s/%s")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create bot api: %w", err)
	}
	return &MessagePublisher{chatID: chatID, tracer: tracer, bot: bot}, nil
}

func (e MessagePublisher) Publish(ctx context.Context, message string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, span := e.tracer.Start(ctx, "MessagePublisher.Publish")
		defer span.End()
		msg := tgbotapi.NewMessage(int64(e.chatID), message)
		_, err := e.bot.Send(msg)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.SetStatus(codes.Ok, "message published")
		return nil
	}
}

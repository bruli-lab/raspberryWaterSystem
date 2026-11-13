package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli/raspberryWaterSystem/internal/app"
	"github.com/bruli/raspberryWaterSystem/internal/domain/weather"
)

func InitStatus(ctx context.Context, qh cqs.QueryHandler, ch cqs.CommandHandler, now time.Time) error {
	weath, err := findWeather(ctx, qh)
	if err != nil {
		return err
	}
	_, err = ch.Handle(ctx, app.CreateStatusCmd{StartedAt: now, Weather: weath})
	return err
}

func UpdateStatus(ctx context.Context, qh cqs.QueryHandler, ch cqs.CommandHandler) error {
	weath, err := findWeather(ctx, qh)
	if err != nil {
		return err
	}
	_, err = ch.Handle(ctx, app.UpdateStatusCmd{Weather: weath})
	return err
}

func findWeather(ctx context.Context, qh cqs.QueryHandler) (weather.Weather, error) {
	result, err := qh.Handle(ctx, app.FindWeatherQuery{})
	if err != nil {
		return weather.Weather{}, err
	}
	weath, ok := result.(weather.Weather)
	if !ok {
		return weather.Weather{}, UnexpectedQueryResultError{result: result}
	}
	return weath, nil
}

type UnexpectedQueryResultError struct {
	result any
}

func (u UnexpectedQueryResultError) Error() string {
	return fmt.Sprintf("unexpected query result of type %T", u.result)
}

package fixtures

import (
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"github.com/bruli/raspberryWaterSystem/internal/domain/weather"
)

type StatusBuilder struct {
	SystemStartedAt *time.Time
	Weather         *weather.Weather
	UpdatedAt       *time.Time
	Active          bool
	Clean           bool
}

func (b StatusBuilder) Build() status.Status {
	var st status.Status
	start := setData(time.Now(), b.SystemStartedAt)
	weath := setData(WeatherBuilder{}.Build(), b.Weather)
	updateAt := setData(start.Add(3*time.Hour), b.UpdatedAt)

	st.Hydrate(start, weath, &updateAt, b.Active, LightBuilder{}.Build(), b.Clean)
	return st
}

type LightBuilder struct {
	Sunrise *time.Time
	Sunset  *time.Time
}

func (b LightBuilder) Build() *status.Light {
	sunrise := time.Date(2020, time.April, 14, 7, 45, 0, 0, time.UTC)
	sunset := time.Date(2020, time.April, 14, 17, 30, 0, 0, time.UTC)
	li, _ := status.NewLight(sunrise, sunset)
	return li
}

package status

import (
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/weather"
)

type Status struct {
	systemStartedAt time.Time
	weather         weather.Weather
	updatedAt       *time.Time
	active          bool
	light           *Light
	clean           bool
}

func (s *Status) IsClean() bool {
	return s.clean
}

func (s *Status) Light() *Light {
	return s.light
}

func (s *Status) IsActive() bool {
	return s.active
}

func (s *Status) SystemStartedAt() time.Time {
	return s.systemStartedAt
}

func (s *Status) Weather() weather.Weather {
	return s.weather
}

func (s *Status) UpdatedAt() *time.Time {
	return s.updatedAt
}

func (s *Status) Update(w weather.Weather, light *Light) {
	s.weather = w
	now := time.Now()
	s.updatedAt = &now
	s.light = light
}

func HydrateStatus(start time.Time, we weather.Weather, updated *time.Time, active bool, light *Light, clean bool) Status {
	return Status{
		systemStartedAt: start,
		weather:         we,
		updatedAt:       updated,
		active:          active,
		light:           light,
		clean:           clean,
	}
}

func (s *Status) Activate() {
	s.active = true
}

func (s *Status) Deactivate() {
	s.active = false
}

func (s *Status) Cleaning() {
	s.clean = true
}

func (s *Status) DoneCleaning() {
	s.clean = false
}

func New(systemStartedAt time.Time, w weather.Weather, light *Light) *Status {
	return &Status{
		systemStartedAt: systemStartedAt,
		weather:         w,
		active:          true,
		light:           light,
		clean:           false,
	}
}

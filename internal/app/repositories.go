package app

import (
	"context"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/weather"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"

	"github.com/bruli/raspberryWaterSystem/internal/domain/status"

	"github.com/bruli/raspberryWaterSystem/internal/domain/zone"
)

//go:generate go tool moq -out zmock_repositories_test.go -pkg app_test . ZoneRepository TemperatureRepository RainRepository StatusRepository ProgramRepository WeeklyProgramRepository TemperatureProgramRepository ExecutionLogRepository LightRepository FertilizerZoneRepository

type ZoneRepository interface {
	FindAll(ctx context.Context) ([]*zone.Zone, error)
	FindByID(ctx context.Context, id string) (*zone.Zone, error)
	Save(ctx context.Context, zo *zone.Zone) error
	Remove(ctx context.Context, zo *zone.Zone) error
	Update(ctx context.Context, zo *zone.Zone) error
}

type TemperatureRepository interface {
	Find(ctx context.Context) (weather.Temperature, weather.Humidity, error)
}

type RainRepository interface {
	Find(ctx context.Context) (bool, error)
}

type StatusRepository interface {
	Save(ctx context.Context, st status.Status) error
	Find(ctx context.Context) (status.Status, error)
	Update(ctx context.Context, st status.Status) error
}

type ProgramRepository interface {
	Save(ctx context.Context, programs *program.Program) error
	FindAll(ctx context.Context) ([]program.Program, error)
	FindByHour(ctx context.Context, hour *program.Hour) (*program.Program, error)
	Remove(ctx context.Context, hour *program.Hour) error
}

type WeeklyProgramRepository interface {
	Save(ctx context.Context, program *program.Weekly) error
	FindAll(ctx context.Context) ([]program.Weekly, error)
	FindByDay(ctx context.Context, day *program.WeekDay) (*program.Weekly, error)
	FindByDayAndHour(ctx context.Context, day *program.WeekDay, hour *program.Hour) (*program.Weekly, error)
	Remove(ctx context.Context, day *program.WeekDay) error
}

type TemperatureProgramRepository interface {
	Save(ctx context.Context, program *program.Temperature) error
	FindByTemperature(ctx context.Context, temperature float32) (*program.Temperature, error)
	Remove(ctx context.Context, temperature float32) error
	FindAll(ctx context.Context) ([]program.Temperature, error)
	FindByTemperatureAndHour(ctx context.Context, temperature float32, hour program.Hour) (program.Temperature, error)
}

type ExecutionLogRepository interface {
	Save(ctx context.Context, logs []program.ExecutionLog) error
	FindAll(ctx context.Context) ([]program.ExecutionLog, error)
}

type LightRepository interface {
	Find(ctx context.Context, date time.Time) (*status.Light, error)
}

type FertilizerZoneRepository interface {
	FindByID(ctx context.Context, id string) (*zone.FertilizerZone, error)
	Save(ctx context.Context, zo *zone.FertilizerZone) error
}

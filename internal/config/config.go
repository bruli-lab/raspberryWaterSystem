package config

import (
	"github.com/caarlos0/env"
)

type Environment struct {
	Value string `env:"ENVIRONMENT,required"`
}

type Config struct {
	ServerURL               string `env:"SERVER_URL,required"`
	Environment             EnvironmentType
	ZonesFile               string `env:"ZONES_FILE,required"`
	AuthToken               string `env:"AUTH_TOKEN,required"`
	DailyProgramsFile       string `env:"DAILY_PROGRAMS_FILE,required"`
	OddProgramsFile         string `env:"ODD_PROGRAMS_FILE,required"`
	EvenProgramsFile        string `env:"EVEN_PROGRAMS_FILE,required"`
	WeeklyProgramsFile      string `env:"WEEKLY_PROGRAMS_FILE,required"`
	TemperatureProgramsFile string `env:"TEMPERATURE_PROGRAMS_FILE,required"`
	ExecutionLogsFile       string `env:"EXECUTION_LOGS_FILE,required"`
	TelegramToken           string `env:"TELEGRAM_TOKEN,required"`
	TelegramChatID          int    `env:"TELEGRAM_CHAT_ID,required"`
	TelegramBotEnabled      bool   `env:"TELEGRAM_BOT_ENABLED,required"`
	NatsServerURL           string `env:"NATS_SERVER_URL,required"`
	EventsDirectory         string `env:"EVENTS_DIRECTORY,required"`
}

func New() (*Config, error) {
	var (
		co Config
		e  Environment
	)

	if err := env.Parse(&co); err != nil {
		return nil, err
	}

	if err := env.Parse(&e); err != nil {
		return nil, err
	}
	envir, err := ParseEnvironment(e.Value)
	if err != nil {
		return nil, err
	}
	co.Environment = envir
	return &co, nil
}

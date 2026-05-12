package fixtures

import (
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
)

type ProgramBuilder struct {
	Hour       *program.Hour
	Executions []program.Execution
	Fertilizer bool
}

func (b ProgramBuilder) Build() program.Program {
	hour, _ := program.ParseHour("15:10")
	if b.Hour != nil {
		hour = *b.Hour
	}
	executions := []program.Execution{
		ExecutionBuilder{}.Build(),
	}
	if b.Executions != nil {
		executions = b.Executions
	}
	return program.HydrateProgram(hour, b.Fertilizer, executions)
}

type FertilizerProgramBuilder struct {
	Hour       *program.Hour
	Executions []program.Execution
	Fertilizer bool
}

func (b FertilizerProgramBuilder) Build() program.FertilizerProgram {
	var pr program.FertilizerProgram
	hour, _ := program.ParseHour("15:10")
	if b.Hour != nil {
		hour = *b.Hour
	}
	executions := []program.Execution{
		ExecutionBuilder{}.Build(),
	}
	if b.Executions != nil {
		executions = b.Executions
	}
	pr.Hydrate(hour, b.Fertilizer, executions)
	return pr
}

type WeeklyBuilder struct {
	WeekDay  *program.WeekDay
	Programs []program.Program
}

func (b WeeklyBuilder) Build() program.Weekly {
	day := program.WeekDay(time.Friday)
	if b.WeekDay != nil {
		day = *b.WeekDay
	}
	programs := []program.Program{
		ProgramBuilder{}.Build(),
	}
	if b.Programs != nil {
		programs = b.Programs
	}
	return program.HydrateWeekly(day, programs)
}

type ExecutionBuilder struct {
	Seconds *program.Seconds
	Zones   []string
}

func (b ExecutionBuilder) Build() program.Execution {
	sec := program.HydrateSeconds(20)
	if b.Seconds != nil {
		sec = *b.Seconds
	}
	zones := []string{
		"1",
	}
	if b.Zones != nil {
		zones = b.Zones
	}
	return program.HydrateExecution(sec, zones)
}

type TemperatureBuilder struct {
	Temperature *float32
	Programs    []program.Program
}

func (b TemperatureBuilder) Build() program.Temperature {
	temperature := float32(28.3)
	if b.Temperature != nil {
		temperature = *b.Temperature
	}
	programs := []program.Program{
		ProgramBuilder{}.Build(),
	}
	if b.Programs != nil {
		programs = b.Programs
	}
	return program.HydrateTemperature(temperature, programs)
}

type ExecutionLogBuilder struct {
	Seconds    *program.Seconds
	ZoneName   *string
	ExecutedAt *time.Time
}

func (b ExecutionLogBuilder) Build() program.ExecutionLog {
	seconds := program.HydrateSeconds(20)
	if b.Seconds != nil {
		seconds = *b.Seconds
	}
	zoneName := "zone test"
	if b.ZoneName != nil {
		zoneName = *b.ZoneName
	}
	executedAt := time.Now()
	if b.ExecutedAt != nil {
		executedAt = *b.ExecutedAt
	}
	return program.HydrateExecutionLog(seconds, zoneName, executedAt)
}

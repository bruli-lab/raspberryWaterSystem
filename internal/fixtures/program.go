package fixtures

import (
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/program"
)

type ProgramBuilder struct {
	Hour       *program.Hour
	Executions []program.Execution
}

func (b ProgramBuilder) Build() program.Program {
	var pr program.Program
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
	pr.Hydrate(hour, executions)
	return pr
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
	var week program.Weekly
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
	week.Hydrate(day, programs)
	return week
}

type ExecutionBuilder struct {
	Seconds *program.Seconds
	Zones   []string
}

func (b ExecutionBuilder) Build() program.Execution {
	var ex program.Execution
	sec, _ := program.ParseSeconds(20)
	if b.Seconds != nil {
		sec = *b.Seconds
	}
	zones := []string{
		"1",
	}
	if b.Zones != nil {
		zones = b.Zones
	}
	ex.Hydrate(sec, zones)
	return ex
}

type TemperatureBuilder struct {
	Temperature *float32
	Programs    []program.Program
}

func (b TemperatureBuilder) Build() program.Temperature {
	var temp program.Temperature
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
	temp.Hydrate(temperature, programs)
	return temp
}

type ExecutionLogBuilder struct {
	Seconds    *program.Seconds
	ZoneName   *string
	ExecutedAt *time.Time
}

func (b ExecutionLogBuilder) Build() program.ExecutionLog {
	var exec program.ExecutionLog
	seconds := program.Seconds(20 * time.Second)
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
	exec.Hydrate(seconds, zoneName, executedAt)
	return exec
}

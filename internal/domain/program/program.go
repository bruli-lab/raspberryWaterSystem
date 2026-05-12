package program

import (
	"errors"
)

var (
	ErrEmptyPrograms       = errors.New("empty programs")
	ErrEmptyExecutionZones = errors.New("empty execution zones")
	ErrInvalidTemperature  = errors.New("invalid temperature")
)

type FertilizerProgram struct {
	hour       Hour
	fertilizer bool
	executions []Execution
}

func (p *FertilizerProgram) Hour() Hour {
	return p.hour
}

func (p *FertilizerProgram) IsFertilizer() bool {
	return p.fertilizer
}

func (p *FertilizerProgram) Executions() []Execution {
	return p.executions
}

func (p *FertilizerProgram) validate() error {
	if len(p.executions) == 0 {
		return ErrEmptyPrograms
	}
	return nil
}

func (p *FertilizerProgram) Hydrate(hour Hour, fertilizer bool, executions []Execution) {
	p.hour = hour
	p.fertilizer = fertilizer
	p.executions = executions
}

func NewFertilizerProgram(hour Hour, fertilizer bool, executions []Execution) (*FertilizerProgram, error) {
	fp := FertilizerProgram{hour: hour, fertilizer: fertilizer, executions: executions}
	if err := fp.validate(); err != nil {
		return nil, err
	}
	return &fp, nil
}

type Program struct {
	hour       Hour
	fertilizer bool
	executions []Execution
}

func (p *Program) Hour() Hour {
	return p.hour
}

func (p *Program) IsFertilizer() bool {
	return p.fertilizer
}

func (p *Program) Executions() []Execution {
	return p.executions
}

func (p *Program) validate() error {
	if len(p.executions) == 0 {
		return ErrEmptyPrograms
	}
	return nil
}

func HydrateProgram(hour Hour, fertilizer bool, executions []Execution) Program {
	return Program{hour: hour, fertilizer: fertilizer, executions: executions}
}

func HydrateProgramWithoutFertilizer(hour Hour, executions []Execution) Program {
	return HydrateProgram(hour, false, executions)
}

func NewWithoutFertilizer(hour Hour, executions []Execution) (*Program, error) {
	return New(hour, false, executions)
}

func New(hour Hour, fertilizer bool, executions []Execution) (*Program, error) {
	pr := Program{hour: hour, fertilizer: fertilizer, executions: executions}
	if err := pr.validate(); err != nil {
		return nil, err
	}
	return &pr, nil
}

type Weekly struct {
	weekDay  WeekDay
	programs []Program
}

func (w *Weekly) WeekDay() WeekDay {
	return w.weekDay
}

func (w *Weekly) Programs() []Program {
	return w.programs
}

func NewWeekly(weekDay WeekDay, programs []Program) (Weekly, error) {
	if len(programs) == 0 {
		return Weekly{}, ErrEmptyPrograms
	}
	return Weekly{weekDay: weekDay, programs: programs}, nil
}

func HydrateWeekly(weekDay WeekDay, programs []Program) Weekly {
	return Weekly{weekDay: weekDay, programs: programs}
}

func (w *Weekly) Update(programs []Program) {
	w.programs = programs
}

type Temperature struct {
	temperature float32
	programs    []Program
}

func (t *Temperature) Temperature() float32 {
	return t.temperature
}

func (t *Temperature) Programs() []Program {
	return t.programs
}

func (t *Temperature) validate() error {
	if len(t.programs) == 0 {
		return ErrEmptyPrograms
	}
	if 0 > t.temperature || 50 < t.temperature {
		return ErrInvalidTemperature
	}
	return nil
}

func NewTemperature(temperature float32, programs []Program) (*Temperature, error) {
	temp := Temperature{temperature: temperature, programs: programs}
	if err := temp.validate(); err != nil {
		return nil, err
	}
	return &temp, nil
}

func HydrateTemperature(temperature float32, programs []Program) Temperature {
	return Temperature{temperature: temperature, programs: programs}
}

func (t *Temperature) Update(programs []Program) {
	t.programs = programs
}

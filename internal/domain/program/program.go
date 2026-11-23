package program

import (
	"errors"
)

var (
	ErrEmptyPrograms       = errors.New("empty programs")
	ErrEmptyExecutionZones = errors.New("empty execution zones")
	ErrInvalidTemperature  = errors.New("invalid temperature")
)

type Program struct {
	hour       Hour
	fertilizer bool
	executions []Execution
	status     Status
}

func (p *Program) Status() Status {
	return p.status
}

func (p *Program) SetStatus(s Status) {
	p.status = s
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

func (p *Program) Update(fertilizer bool, executions []Execution, status Status) error {
	updated := Program{hour: p.hour, fertilizer: fertilizer, executions: executions, status: status}
	if err := updated.validate(); err != nil {
		return err
	}
	p.fertilizer = fertilizer
	p.executions = executions
	p.status = status
	return nil
}

func (p *Program) validate() error {
	if len(p.executions) == 0 {
		return ErrEmptyPrograms
	}
	return nil
}

func HydrateProgram(hour Hour, fertilizer bool, executions []Execution, status Status) Program {
	return Program{hour: hour, fertilizer: fertilizer, executions: executions, status: status}
}

func HydrateProgramWithoutFertilizer(hour Hour, executions []Execution, status Status) Program {
	return HydrateProgram(hour, false, executions, status)
}

func NewWithoutFertilizer(hour Hour, executions []Execution) (*Program, error) {
	return New(hour, false, executions)
}

func New(hour Hour, fertilizer bool, executions []Execution) (*Program, error) {
	pr := Program{hour: hour, fertilizer: fertilizer, executions: executions, status: EnabledStatus}
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

func (w *Weekly) Update(programs []Program) error {
	if len(programs) == 0 {
		return ErrEmptyPrograms
	}
	w.programs = programs
	return nil
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

func (t *Temperature) Update(programs []Program) error {
	updated := Temperature{temperature: t.temperature, programs: programs}
	if err := updated.validate(); err != nil {
		return err
	}
	t.programs = programs
	return nil
}

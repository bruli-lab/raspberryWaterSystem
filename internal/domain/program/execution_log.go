package program

import (
	"errors"
	"time"
)

var (
	ErrEmptyZoneName     = errors.New("empty execution log zone name")
	ErrInvalidExecutedAt = errors.New("invalid execution log executed at")
)

type ExecutionLog struct {
	seconds    Seconds
	zoneName   string
	executedAt time.Time
}

func (e ExecutionLog) Seconds() Seconds {
	return e.seconds
}

func (e ExecutionLog) ZoneName() string {
	return e.zoneName
}

func (e ExecutionLog) ExecutedAt() time.Time {
	return e.executedAt
}

func (e ExecutionLog) validate() error {
	if _, err := ParseSeconds(e.seconds.Int()); err != nil {
		return err
	}
	if e.zoneName == "" {
		return ErrEmptyZoneName
	}
	if e.executedAt.IsZero() {
		return ErrInvalidExecutedAt
	}
	return nil
}

func NewExecutionLog(seconds Seconds, zoneName string, executedAt time.Time) (ExecutionLog, error) {
	exec := ExecutionLog{seconds: seconds, zoneName: zoneName, executedAt: executedAt}
	if err := exec.validate(); err != nil {
		return ExecutionLog{}, err
	}
	return exec, nil
}

func HydrateExecutionLog(seconds Seconds, zoneName string, executedAt time.Time) ExecutionLog {
	return ExecutionLog{seconds: seconds, zoneName: zoneName, executedAt: executedAt}
}

package program

import "errors"

const (
	EnabledStatus  Status = "enabled"
	DisabledStatus Status = "disabled"
)

var ErrInvalidStatus = errors.New("invalid program status")

type Status string

func (s Status) String() string {
	return string(s)
}

func (s Status) IsEnabled() bool {
	return s == EnabledStatus
}

func ParseStatus(s string) (Status, error) {
	st := Status(s)
	switch st {
	case EnabledStatus, DisabledStatus:
		return st, nil
	default:
		return "", ErrInvalidStatus
	}
}

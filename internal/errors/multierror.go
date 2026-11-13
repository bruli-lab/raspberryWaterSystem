package errors

import (
	"errors"
	"strings"
)

var _ error = &MultiError{}

const (
	multiErrorPrefix    = "multi error: "
	multiErrorSeparator = "; "
)

func NewMultiError() *MultiError {
	return &MultiError{
		errors: make([]error, 0),
	}
}

type MultiError struct {
	errors []error
}

func (e MultiError) Error() string {
	if len(e.errors) == 0 {
		return ""
	}

	errorMsgs := make([]string, len(e.errors))
	for i, e := range e.errors {
		errorMsgs[i] = e.Error()
	}
	return multiErrorPrefix + strings.Join(errorMsgs, multiErrorSeparator)
}

func (e MultiError) ErrResult() error {
	if len(e.errors) == 0 {
		return nil
	}
	return &e
}

func (e MultiError) Unwrap() error {
	return nil
}

func (e *MultiError) Add(err error) {
	e.errors = append(e.errors, err)
}

func (e MultiError) Is(err error) bool {
	for _, e := range e.errors {
		if errors.Is(e, err) {
			return true
		}
	}
	return false
}

func (e MultiError) Errors() []error {
	return e.errors
}

func (e MultiError) HasErrors() bool {
	return len(e.errors) > 0
}

package errors

import (
	"fmt"
)

type NotFoundError struct {
	id string
}

func NewNotFoundError(id string) NotFoundError {
	return NotFoundError{id: id}
}

func (n NotFoundError) Error() string {
	return fmt.Sprintf("not found %q", n.id)
}

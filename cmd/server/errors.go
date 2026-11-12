package main

import "fmt"

type UnexpectedQueryResultError struct {
	result any
}

func (u UnexpectedQueryResultError) Error() string {
	return fmt.Sprintf("unexpected query result of type %T", u.result)
}

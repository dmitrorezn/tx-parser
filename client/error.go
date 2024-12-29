package client

import (
	"errors"
	"fmt"
)

type Error struct {
	statusCode int
	body       string
}

func NewError(code int, body string) *Error {
	return &Error{
		body:       body,
		statusCode: code,
	}
}

func (e Error) Is(err error) bool {
	target := &Error{}
	if !errors.As(err, &target) {
		return false
	}

	return target.body == e.body && target.statusCode == target.statusCode
}

func (e Error) StatusCode() int {
	return e.statusCode
}
func (e Error) Error() string {
	return fmt.Sprintf("HTTP ERROR code=%d, body=%s", e.statusCode, e.body)
}

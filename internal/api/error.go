package api

import "fmt"

// Error contains rich information about an API failure
type Error struct {
	message string
	status  int
}

var _ error = (*Error)(nil)

// StatusCode returns the HTTP status code of the failure
func (e *Error) StatusCode() int {
	return e.status
}

// Message returns the description of the error
func (e *Error) Message() string {
	return e.message
}

func (e *Error) Error() string {
	return fmt.Sprintf("http error: %s (code: %d)", e.message, e.status)
}

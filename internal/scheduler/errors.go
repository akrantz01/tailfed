package scheduler

import "time"

type retryError struct {
	after time.Duration
	err   error
}

var _ error = (*retryError)(nil)

// Retry a job sooner than the schedule frequency
func Retry(after time.Duration, err error) error {
	return &retryError{after, err}
}

func (r *retryError) Error() string {
	return r.err.Error()
}

func (r *retryError) Unwrap() error {
	return r.err
}

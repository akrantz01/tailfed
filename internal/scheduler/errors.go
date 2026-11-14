package scheduler

type persistentError struct {
	err error
}

var _ error = (*persistentError)(nil)

// Persistent marks an error as unlikely to resolve itself quickly. Instead of backing off exponentially, the job will
// be retried at the maximum backoff interval.
func Persistent(err error) error {
	return &persistentError{err}
}

func (p *persistentError) Error() string {
	return p.err.Error()
}

func (p *persistentError) Unwrap() error {
	return p.err
}

package scheduler

import "context"

// Job is something that can be run repetitively
type Job interface {
	Name() string
	Run(ctx context.Context) error
}

// JobFunc creates a new job from a function
func JobFunc(name string, run func(ctx context.Context) error) Job {
	return &jobFunc{name, run}
}

type jobFunc struct {
	name string
	run  func(ctx context.Context) error
}

func (j *jobFunc) Name() string {
	return j.name
}

func (j *jobFunc) Run(ctx context.Context) error {
	return j.run(ctx)
}

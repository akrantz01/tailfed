package scheduler

import (
	"context"
	"time"

	"github.com/akrantz01/tailfed/internal/logging"
	"github.com/go-co-op/gocron/v2"
	"github.com/jonboulle/clockwork"
	"github.com/sirupsen/logrus"
)

type Scheduler struct {
	logger *logrus.Entry
	inner  gocron.Scheduler
}

// NewScheduler creates a new scheduler to run the refresh job periodically
func NewScheduler(ctx context.Context) (*Scheduler, error) {
	clock := clockwork.FromContext(ctx)
	inner, err := gocron.NewScheduler(
		gocron.WithLogger(logging.NewCronAdapter(nil)),
		gocron.WithClock(clock),
	)
	if err != nil {
		return nil, err
	}

	return &Scheduler{
		inner:  inner,
		logger: logrus.WithField("logger", "scheduler"),
	}, nil
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.inner.Start()
}

// RegisterJob registers a new job with the specified frequency
func (s *Scheduler) RegisterJob(frequency time.Duration, job Job) error {
	j, err := s.inner.NewJob(gocron.DurationJob(frequency), s.toTask(job), gocron.WithName(job.Name()))
	if err != nil {
		return err
	}

	s.logger.WithField("name", job.Name()).Debug("registered new job")
	return j.RunNow()
}

func (s *Scheduler) toTask(job Job) gocron.Task {
	logger := s.logger.WithField("name", job.Name())

	return gocron.NewTask(func(ctx context.Context) {
		logger.Debug("starting job...")
		if err := job.Run(ctx); err != nil {
			logger.WithError(err).Error("job execution failed")
		} else {
			logger.Debug("job execution completed")
		}
	})
}

// Stop finishes any jobs in progress and halt the scheduler
func (s *Scheduler) Stop() error {
	return s.inner.Shutdown()
}

package scheduler

import (
	"context"
	"errors"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/sirupsen/logrus"
)

type JobFn func(ctx context.Context) error

type Scheduler struct {
	logger *logrus.Entry

	clock clockwork.Clock
	loop  clockwork.Ticker
	retry clockwork.Timer

	shutdownCtx      context.Context
	shutdownFn       context.CancelFunc
	shutdownComplete chan struct{}

	jobCtx context.Context
	job    JobFn
}

func NewScheduler(ctx context.Context, frequency time.Duration, job JobFn) *Scheduler {
	clock := clockwork.FromContext(ctx)

	shutdownCtx, shutdownFn := context.WithCancel(context.Background())

	return &Scheduler{
		logger: logrus.WithField("logger", "scheduler"),
		clock:  clock,
		loop:   clock.NewTicker(frequency),
		retry:  clock.NewTimer(0 * time.Second),

		shutdownCtx:      shutdownCtx,
		shutdownFn:       shutdownFn,
		shutdownComplete: make(chan struct{}, 1),

		jobCtx: ctx,
		job:    job,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	go s.scheduler()
}

func (s *Scheduler) scheduler() {
	s.logger.Debug("scheduler started")

	loopC := s.loop.Chan()
	retryC := s.retry.Chan()
	for {
		select {
		case <-retryC:
			s.run()
		case <-loopC:
			s.run()
		case <-s.shutdownCtx.Done():
			s.logger.Debug("scheduler shutdown")
			s.shutdownComplete <- struct{}{}
			return
		}
	}
}

func (s *Scheduler) run() {
	s.logger.Debug("starting job execution")
	if err := s.job(s.jobCtx); err != nil {
		logger := s.logger.WithError(err)

		var retryErr *retryError
		if errors.As(err, &retryErr) {
			s.retry.Reset(retryErr.after)
			logger = logger.WithField("retry", retryErr.after)
		}

		logger.WithError(err).Error("job execution failed")
	} else {
		s.logger.Debug("job execution complete")
	}
}

// Stop finishes any jobs in progress and halt the scheduler
func (s *Scheduler) Stop() {
	s.loop.Stop()
	s.shutdownFn()
	<-s.shutdownComplete
}

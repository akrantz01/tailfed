package scheduler

import (
	"context"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/sirupsen/logrus"
)

type JobFn func(ctx context.Context) error

type Scheduler struct {
	logger *logrus.Entry

	clock  clockwork.Clock
	ticker clockwork.Ticker

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
		ticker: clock.NewTicker(frequency),

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

	s.run()

	c := s.ticker.Chan()
	for {
		select {
		case <-c:
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
		s.logger.WithError(err).Error("job execution failed")
	} else {
		s.logger.Debug("job execution complete")
	}
}

// Stop finishes any jobs in progress and halt the scheduler
func (s *Scheduler) Stop() {
	s.ticker.Stop()
	s.shutdownFn()
	<-s.shutdownComplete
}

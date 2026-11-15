package scheduler

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/jonboulle/clockwork"
	"github.com/sirupsen/logrus"
)

// InitialMaxBackoff caps the delay between retries before the job has succeeded for the first time
const InitialMaxBackoff = 5 * time.Minute

// Schedule describes when a job should next run following a success
type Schedule struct {
	Next time.Time
	Max  time.Time
}

type JobFn func(ctx context.Context) (Schedule, error)

type Scheduler struct {
	logger *logrus.Entry

	clock   clockwork.Clock
	timer   clockwork.Timer
	backoff *backoff.ExponentialBackOff

	deadline time.Time

	immediate chan struct{}

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	job JobFn
}

// NewScheduler creates a scheduler which runs the job immediately once started
func NewScheduler(ctx context.Context, job JobFn) *Scheduler {
	clock := clockwork.FromContext(ctx)
	ctx, cancel := context.WithCancel(ctx)

	b := backoff.NewExponentialBackOff()
	b.MaxInterval = InitialMaxBackoff
	b.Reset()

	return &Scheduler{
		logger:  logrus.WithField("component", "scheduler"),
		clock:   clock,
		timer:   clock.NewTimer(0),
		backoff: b,

		immediate: make(chan struct{}, 1),

		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),

		job: job,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	go s.scheduler()
}

func (s *Scheduler) scheduler() {
	defer close(s.done)
	s.logger.Debug("scheduler started")

	for {
		select {
		case <-s.immediate:
			s.run()
		case <-s.timer.Chan():
			s.run()
		case <-s.ctx.Done():
			s.timer.Stop()
			s.logger.Debug("scheduler shutdown")
			return
		}
	}
}

// RunNow enqueues a run immediately
func (s *Scheduler) RunNow() {
	select {
	case s.immediate <- struct{}{}:
	default:
	}
}

func (s *Scheduler) run() {
	s.logger.Debug("starting job execution")
	schedule, err := s.job(s.ctx)
	if s.ctx.Err() != nil {
		return
	}

	var delay time.Duration
	if err != nil {
		delay = s.failure(err)
		s.logger.WithError(err).WithField("retry", delay).Error("job execution failed")
	} else {
		delay = s.success(schedule)
		s.logger.WithField("next", delay).Debug("job execution complete")
	}

	s.timer.Reset(delay)
}

func (s *Scheduler) success(schedule Schedule) time.Duration {
	delay := schedule.Next.Sub(s.clock.Now())
	s.deadline = schedule.Max

	s.backoff.MaxInterval = delay / 2
	s.backoff.Reset()

	return delay
}

func (s *Scheduler) failure(err error) time.Duration {
	delay := s.backoff.MaxInterval
	if _, ok := errors.AsType[*persistentError](err); !ok {
		delay = min(s.backoff.NextBackOff(), delay)
	}

	if remaining := s.deadline.Sub(s.clock.Now()); remaining > 0 {
		delay = min(delay, remaining)
	}

	return delay
}

// Stop cancels any job in progress and halts the scheduler
func (s *Scheduler) Stop() {
	s.cancel()
	<-s.done
}

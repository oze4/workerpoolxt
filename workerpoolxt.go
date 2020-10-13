package workerpoolxt

import (
	"context"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/gammazero/workerpool"
)

// New creates *WorkerPoolXT
func New(maxWorkers int, defaultJobTimeout time.Duration) *WorkerPoolXT {
	w := &WorkerPoolXT{
		WorkerPool:     workerpool.New(maxWorkers),
		defaultTimeout: defaultJobTimeout,
		responsesChan:  make(chan Response),
		killswitch:     make(chan bool),
	}

	go w.processResponses()
	return w
}

// WorkerPoolXT extends `github.com/gammazero/workerpool`
type WorkerPoolXT struct {
	*workerpool.WorkerPool
	defaultTimeout time.Duration
	killswitch     chan bool
	options        Options
	once           sync.Once
	responsesChan  chan Response
	responses      []Response
}

// SubmitXT submits a job. Allows you to not
// only submit a job, but get the response from it
func (wp *WorkerPoolXT) SubmitXT(job Job) {
	wp.Submit(wp.wrap(job))
}

// StopWaitXT gets results then kills the worker pool.
// You cannot add jobs after calling `StopWaitXT()`
func (wp *WorkerPoolXT) StopWaitXT() (rs []Response) {
	wp.stop(false)
	return wp.responses
}

// WithOptions sets default options for each job
// You can also supply options on a per job basis,
// which will override the default options.
func (wp *WorkerPoolXT) WithOptions(o Options) {
	wp.options = o
}

// getBackoff determines if a job is using Retry - if it is
// we configure the backoff
//
// TODO... I would love a 'native' way to retry jobs, though!
//
func (wp *WorkerPoolXT) getBackoff(job Job) backoff.BackOff {
	var rbo backoff.BackOff
	if job.Retry > 0 {
		rbo = backoff.WithMaxRetries(backoff.NewExponentialBackOff(), uint64(job.Retry))
	}
	return rbo
}

// getOptions decides which options to use : default or job
func (wp *WorkerPoolXT) getOptions(job Job) Options {
	if job.Options != nil {
		return job.Options
	}
	if wp.options != nil {
		return wp.options
	}
	return make(Options)
}

// getTimeout decides which timeout to use : default or job
func (wp *WorkerPoolXT) getTimeout(job Job) time.Duration {
	if job.Timeout != 0 {
		return job.Timeout
	}
	return wp.defaultTimeout
}

// processResponses listens for anything on
// the responses chan and aggregates results
func (wp *WorkerPoolXT) processResponses() {
	for {
		select {
		case response, ok := <-wp.responsesChan:
			if !ok {
				goto Done
			}
			wp.responses = append(wp.responses, response)
		}
	}
Done:
	<-wp.killswitch
}

// stop either stops the worker pool now or later
func (wp *WorkerPoolXT) stop(now bool) {
	wp.once.Do(func() {
		if now {
			wp.Stop()
		} else {
			wp.StopWait()
		}
		close(wp.responsesChan)
		wp.killswitch <- true
	})
}

// work should be ran on it's own goroutine. Sorts out job metadata and options and runs user provided Job.Task
func (wp *WorkerPoolXT) work(ctx context.Context, done context.CancelFunc, j Job, ts time.Time, bo backoff.BackOff) {
	jobtask := func() error {
		o := wp.getOptions(j)
		// Run job provided by caller
		f := j.Task(o)
		// We only want to return an error (or nil if no error)
		// if the job is using Retry (which ultimately calls `backoff`).
		// This is how `backoff` knows the job failed, but if we are
		// not using Retry it will cause errors without this check.
		if f.Error != nil && bo != nil {
			return f.Error
		}

		f.runtimeDuration = time.Since(ts)
		f.name = j.Name

		// This check is important as it keeps
		// from sending duplicate responses on our responses chan
		// ** See `./doc.go`, section `Response Channel` for more info **
		if ctx.Err() == nil {
			wp.responsesChan <- f
		}
		return nil
	}

	if bo != nil {
		// If the job is using Retry
		jobErr := backoff.Retry(jobtask, bo)
		if jobErr != nil {
			// Send a Response, with our error, to our response chan.
			wp.responsesChan <- Response{name: j.Name, Error: jobErr}
		}
	} else {
		// If job is not using Retry,
		// simply call the job we were given
		jobtask()
	}
	// context.CancelFunc
	done()
}

// wrap generates the func that we pass to Submit.
func (wp *WorkerPoolXT) wrap(job Job) func() {
	timeout := wp.getTimeout(job)

	// This is the func() that ultimately
	// gets passed to `workerpool.Submit(f)`
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		start := time.Now()
		retryBackoff := wp.getBackoff(job)

		go wp.work(ctx, cancel, job, start, retryBackoff)

		select {
		case <-ctx.Done():
			switch ctx.Err() {
			case context.DeadlineExceeded:
				// If our timeout has passed, return an error object,
				// disregarding any response from job which timed out.
				// ** See `./doc.go`, section `Response Channel` for more info **
				wp.responsesChan <- Response{
					Error:           context.DeadlineExceeded,
					name:            job.Name,
					runtimeDuration: time.Since(start),
				}
			}
		}
	}
}

// Options hold misc options
type Options map[string]interface{}

// Job holds job data
type Job struct {
	Name    string
	Task    func(Options) Response
	Timeout time.Duration
	Options Options
	Retry   int
}

// Response holds job results
type Response struct {
	Error           error
	Data            interface{}
	name            string
	runtimeDuration time.Duration
}

// RuntimeDuration returns the amount of time it took to run the job
func (r *Response) RuntimeDuration() time.Duration {
	return r.runtimeDuration
}

// Name returns the job name
func (r *Response) Name() string {
	return r.name
}

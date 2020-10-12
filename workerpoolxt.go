package workerpoolxt

import (
	"context"
	"sync"
	"time"

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

// SubmitXT submits a job. Allows you to not only submit a job, but get the response from it
func (wp *WorkerPoolXT) SubmitXT(job Job) {
	wp.Submit(wp.wrap(job))
}

// StopWaitXT gets results then kills the worker pool. You cannot add jobs after calling `StopWaitXT()`
func (wp *WorkerPoolXT) StopWaitXT() (rs []Response) {
	wp.stop(false)
	return wp.responses
}

// WithOptions sets default options for each job You can also supply options on a per job basis,
// which will override the default options.
func (wp *WorkerPoolXT) WithOptions(o Options) {
	wp.options = o
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

// processResponses listens for anything on the responses chan and aggregates results
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

// wrap generates the func that we pass to Submit.
// - If a timeout is not supplied with the job, we use the global default supplied when `New()` is called
// - If optoins are not supplied with the job, we use the global default supplied when `New()` is called
// - Responsible for injecting timeout and runtime duration
func (wp *WorkerPoolXT) wrap(job Job) func() {
	timeout := wp.getTimeout(job)

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		start := time.Now()

		go func(ctx context.Context, done context.CancelFunc, j Job, ts time.Time) {
			o := wp.getOptions(j)
			// Run user provided task & set job specific metadata
			f := j.Task(o)
			f.runtimeDuration = time.Since(ts)
			f.name = j.Name
			// This check is important as it keeps from sending duplicate responses on our responses chan
			if ctx.Err() == nil {
				wp.responsesChan <- f
			}
			done()
		}(ctx, cancel, job, start)

		select {
		// If our timeout has passed, return an error object, disregarding any response from job which timed out
		case <-ctx.Done():
			switch ctx.Err() {
			case context.DeadlineExceeded:
				wp.responsesChan <- Response{
					Error:           context.DeadlineExceeded,
					name:            job.Name,
					runtimeDuration: time.Since(start),
				}
			}
		}
	}
}

// getTimeout decides which timeout to use : default or job
func (wp *WorkerPoolXT) getTimeout(job Job) time.Duration {
	if job.Timeout != 0 {
		return job.Timeout
	}
	return wp.defaultTimeout
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

// Options hold misc options
type Options map[string]interface{}

// Job holds job data
type Job struct {
	Name    string
	Task    func(Options) Response
	Timeout time.Duration
	Options Options
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

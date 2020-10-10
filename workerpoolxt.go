package workerpoolxt

import (
	"context"
	"time"

	"github.com/gammazero/workerpool"
)

var (
	defaultResponsesBufferSize = 100 // default buffer size for our results chan
)

// New creates *WorkerPoolXT
func New(maxWorkers int, defaultJobTimeout time.Duration) *WorkerPoolXT {
	return &WorkerPoolXT{
		WorkerPool:     workerpool.New(maxWorkers),
		defaultTimeout: defaultJobTimeout,
		responses:      make(chan Response, defaultResponsesBufferSize),
	}
}

// WorkerPoolXT extends `github.com/gammazero/workerpool` by:
//  - Collects the output of all jobs so you can work with it later
//  - Job runtime duration stats baked in
//  - Job timeouts; global timeout or per job basis
//  - Lets you "pause" to gather results at any given time
type WorkerPoolXT struct {
	*workerpool.WorkerPool
	count          int           // Job count
	resultCount    int           // Job result count
	defaultTimeout time.Duration // Job timeout
	responses      chan Response // Job results
}

// SubmitXT submits a job
// Allows you to not only submit a job, but get the response from it
func (r *WorkerPoolXT) SubmitXT(job Job) {
	r.Submit(r.wrap(job))
}

// SubmitAllXT allows you to supply multiple Events
func (r *WorkerPoolXT) SubmitAllXT(jobs []Job) {
	for _, job := range jobs {
		r.Submit(r.wrap(job))
	}
}

// StopWaitXT gets results then kills the worker pool
// - You cannot add jobs after calling `StopWaitXT()``
// - Wrapper for `workerpool.StopWait()`
func (r *WorkerPoolXT) StopWaitXT() (rs []Response) {
	r.StopWait()
	close(r.responses)
	for response := range r.responses {
		rs = append(rs, response)
	}
	return rs
}

func (r *WorkerPoolXT) resetCounters() {
	r.count = 0
	r.resultCount = 0
}

// wrap generates the func that we pass to Submit.
// - If a timeout is not supplied with the job, we use the global default supplied when `New()` is called
// - Responsible for injecting timeout and runtime duration
func (r *WorkerPoolXT) wrap(job Job) func() {
	r.count++
	timeout := r.defaultTimeout
	if job.Timeout != 0 {
		timeout = job.Timeout
	}

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		start := time.Now()

		go func(ctx context.Context, done context.CancelFunc, job Job, timerStart time.Time) {
			j := job.Task()
			j.runtimeDuration = time.Since(timerStart)
			j.name = job.Name
			// This check is important as it keeps from sending duplicate responses on our responses chan
			if ctx.Err() == nil {
				r.responses <- j
			}
			done()
		}(ctx, cancel, job, start)

		select {
		case <-ctx.Done():
			switch ctx.Err() {
			case context.DeadlineExceeded:
				r.responses <- Response{
					Error:           context.DeadlineExceeded,
					name:            job.Name,
					runtimeDuration: time.Since(start),
				}
			}
		}
	}
}

// Job holds job data
type Job struct {
	Name    string
	Task    func() Response
	Timeout time.Duration
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

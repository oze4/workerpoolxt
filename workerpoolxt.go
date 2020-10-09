package workerpoolxt

import (
	"context"
	"time"

	"github.com/gammazero/workerpool"
)

var (
	// defaultResponsesBufferSize is the default buffer size for our results chan
	defaultResponsesBufferSize = 100
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
//  - `myworkerpoolxt.WaitXT()` lets you "pause" to gather job output at any given time,
//    after which you may continue to submit jobs to the workerpool
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
func (r *WorkerPoolXT) StopWaitXT() []Response {
	r.StopWait()
	close(r.responses)

	var responses []Response
	for response := range r.responses {
		responses = append(responses, response)
	}

	return responses
}

// WaitXTExperimental is an experimental func that needs to be test still.. just an idea I had
//  - "pauses" the workerpool to get all current and pending event reactions
//  - Once we have all job responses, we return them. You can continue to use the workerpool
//  - Unlike `workerpool.StopWait()` or `workerpoolxt.StopWaitXT()` this does not kill the worker pool,
//    meaning, you can continue to add jobs after
func (r *WorkerPoolXT) WaitXTExperimental() []Response {
	var responses []Response

	for {
		select {
		case response := <-r.responses:
			responses = append(responses, response)
			r.resultCount++
			if r.count == r.resultCount {
				goto Return
			}
		}
	}

Return:
	r.resetCounters()
	return responses
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
					Error: context.DeadlineExceeded,
					name:  job.Name,
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

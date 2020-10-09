package workerpoolxt

import (
	"context"
	"time"

	"github.com/gammazero/workerpool"
)

// New creates a new WorkerPoolXT
func New(maxWorkers int, defaultJobTimeout time.Duration) *WorkerPoolXT {
	return &WorkerPoolXT{
		WorkerPool:     workerpool.New(maxWorkers),
		defaultTimeout: defaultJobTimeout,
		responses:      make(chan Response, 1000),
	}
}

// WorkerPoolXT extends `github.com/gammazero/workerpool`
type WorkerPoolXT struct {
	*workerpool.WorkerPool
	count          int           // Job count
	resultCount    int           // Job result count
	defaultTimeout time.Duration // Job timeout
	responses      chan Response
}

// SubmitXT submits a job
// Allows you to not only submit a job, but get the response
// from it
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
//
// You cannot add jobs after calling `ReactionsStop()``
func (r *WorkerPoolXT) StopWaitXT() []Response {
	r.StopWait()
	close(r.responses)

	var responses []Response
	for response := range r.responses {
		responses = append(responses, response)
	}

	return responses
}

// WaitXT **SHOULD NOT BE USED YET** I still need to do more testing
//
// Essentially, WaitXT "pauses" the workerpool to get all current
// and pending event reactions. Once we have all reactions we return them
// and you can continue to use the workerpool.
//
// Unlike `ReactionsStop()` this does not kill the worker pool. You can continue
// to add jobs (events) after calling `ReactionsWait()`
func (r *WorkerPoolXT) WaitXT() []Response {
	var responses []Response

	for {
		select {
		case response := <-r.responses:
			responses = append(responses, response)
			r.resultCount++
			if (r.count) == (r.resultCount) {
				goto Return
			}
		}
	}

Return:
	return responses
}

// worker kicks off job and places result on results chan unless timeout is exceeded. If that is
// the case we do nothing with return and let `wrapper` handle context deadline exceeded
func (r *WorkerPoolXT) work(ctx context.Context, done context.CancelFunc, job Job, start time.Time) {
	j := job.Task()
	j.runtimeDuration = time.Since(start)
	j.name = job.Name

	if ctx.Err() == nil {
		r.responses <- j
	}

	done()
}

// wrap should be private
func (r *WorkerPoolXT) wrap(job Job) func() {
	timeout := r.defaultTimeout
	// If Job contains a Timeout use it, otherwise use the general timeout
	if job.Timeout != 0 {
		timeout = job.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	r.count++

	return func() {
		start := time.Now()
		go r.work(ctx, cancel, job, start)

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
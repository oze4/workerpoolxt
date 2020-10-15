package workerpoolxt

import (
	"context"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/gammazero/workerpool"
)

// New creates *WorkerPoolXT
func New(ctx context.Context, maxWorkers int) *WorkerPoolXT {
	w := &WorkerPoolXT{
		WorkerPool:    workerpool.New(maxWorkers),
		context:       ctx,
		responsesChan: make(chan Response),
		killswitch:    make(chan bool),
	}
	go w.processResponses()
	return w
}

// WorkerPoolXT extends `github.com/gammazero/workerpool`
type WorkerPoolXT struct {
	*workerpool.WorkerPool
	context       context.Context
	killswitch    chan bool
	options       Options
	once          sync.Once
	responsesChan chan Response
	responses     []Response
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
// TODO
// I would love a 'native' way to retry jobs, though!
func (wp *WorkerPoolXT) getBackoff(job Job) backoff.BackOff {
	var rbo backoff.BackOff
	if job.Retry > 0 {
		rbo = backoff.WithMaxRetries(
			backoff.NewExponentialBackOff(),
			uint64(job.Retry),
		)
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

// getContext decides which context to use : default or job
func (wp *WorkerPoolXT) getContext(job Job) (context.Context, context.CancelFunc) {
	if job.Context != nil {
		return context.WithCancel(job.Context)
	}
	return context.WithCancel(wp.context)
}

// getResult reacts to a jobs context
func (wp *WorkerPoolXT) getResult(job Job) {
	select {
	case <-job.context.Done():
		switch job.context.Err() {
		case context.DeadlineExceeded:
			wp.responsesChan <- newResponseError(context.DeadlineExceeded, job.Name, job.startTime)
		case context.Canceled:
			if !job.contextCancelledInternally {
				wp.responsesChan <- newResponseError(context.Canceled, job.Name, job.startTime)
			}
		}
	}
}

// finalize returns the final func that calls the user provided Job.Task
func (wp *WorkerPoolXT) finalize(job Job) func() error {
	job.backoff = wp.getBackoff(job)

	return func() error {
		o := wp.getOptions(job)
		f := job.Task(o)

		// If Job not using Retry, don't want to return an error
		if f.Error != nil && job.backoff != nil {
			return f.Error
		}

		// No point in setting this before checking for error
		f.runtimeDuration = time.Since(job.startTime)
		f.name = job.Name

		// Only send a response if our contex is good
		if job.context.Err() == nil {
			wp.responsesChan <- f
		}

		// We can return nil whether Job is using Retry or not
		return nil
	}
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

// start should be ran on it's own goroutine. Sorts out job metadata and options and runs user provided Job.Task
func (wp *WorkerPoolXT) do(job Job) {
	final := wp.finalize(job)
	todo := func() {
		final()
	}

	// If the job is using Retry, change our `theWork` func
	if job.backoff != nil {
		todo = func() {
			jobErr := backoff.Retry(final, job.backoff)
			if jobErr != nil {
				wp.responsesChan <- Response{
					name:  job.Name,
					Error: jobErr,
				}
			}
		}
	}

	todo()
	job.successCancel() // Prob need a better way to do this
}

// wrap generates the func that we pass to Submit.
func (wp *WorkerPoolXT) wrap(job Job) func() {
	jobctx, jobdone := wp.getContext(job)

	// This is the func() that ultimately gets passed to `workerpool.Submit(f)`
	return func() {
		job.request = &request{
			context:    jobctx,
			cancelFunc: jobdone,
			startTime:  time.Now(),
		}

		go wp.do(job)
		wp.getResult(job)
	}
}

// Options hold misc options
type Options map[string]interface{}

// Job holds job data
type Job struct {
	*request
	Name    string
	Task    func(Options) Response
	Context context.Context
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

// request holds misc data for each job
type request struct {
	context                    context.Context
	cancelFunc                 context.CancelFunc
	backoff                    backoff.BackOff
	job                        Job
	startTime                  time.Time
	contextCancelledInternally bool
}

// successCancel cancels our context and sets isSuccess flag
func (r *request) successCancel() {
	r.contextCancelledInternally = true
	r.cancelFunc()
}

// helper func
func newResponseError(err error, jobName string, start time.Time) Response {
	return Response{
		Error:           err,
		name:            jobName,
		runtimeDuration: time.Since(start),
	}
}

package workerpoolxt

import (
	"context"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/gammazero/workerpool"
)

// New creates WorkerPoolXT
func New(ctx context.Context, maxWorkers int) *WorkerPoolXT {
	w := &WorkerPoolXT{
		WorkerPool:    workerpool.New(maxWorkers),
		context:       ctx,
		responsesChan: make(chan Response),
		killswitch:    make(chan struct{}),
	}

	go w.processResponses()

	return w
}

// WorkerPoolXT extends `github.com/gammazero/workerpool`
type WorkerPoolXT struct {
	*workerpool.WorkerPool
	context       context.Context
	killswitch    chan struct{}
	options       Options
	once          sync.Once
	responsesChan chan Response
	responses     []Response
}

// SubmitXT submits a job which you can get a response from
func (wp *WorkerPoolXT) SubmitXT(job Job) {
	wp.Submit(wp.wrap(job))
}

// StopWaitXT gets results then kills the worker pool
func (wp *WorkerPoolXT) StopWaitXT() (rs []Response) {
	wp.stop(false)
	return wp.responses
}

// WithOptions sets default options for each job
func (wp *WorkerPoolXT) WithOptions(o Options) {
	wp.options = o
}

// do gets the appropriate payload and does it
func (wp *WorkerPoolXT) do(job Job) {
	payload := wp.newPayload(job)

	todo := func() {
		payload()
	}

	// If the job is using Retry
	if job.backoff != nil {
		todo = func() {
			err := backoff.Retry(payload, job.backoff)
			if err != nil {
				job.response <- newResponseError(err, job.Name, job.startTime)
			}
		}
	}

	todo()
	job.done()
}

// getBackoff determines if a job is using Retry
// TODO I would love a 'native' way to retry jobs, though!
func (wp *WorkerPoolXT) getBackoff(job Job) backoff.BackOff {
	var b backoff.BackOff
	if job.Retry > 0 {
		x, r := backoff.NewExponentialBackOff(), uint64(job.Retry)
		b = backoff.WithMaxRetries(x, r)
	}
	return b
}

// getContext decides which context to use : default or job
func (wp *WorkerPoolXT) getContext(job Job) (context.Context, context.CancelFunc) {
	if job.Context != nil {
		return context.WithCancel(job.Context)
	}
	return context.WithCancel(wp.context)
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

// getResult reacts to a jobs context
func (wp *WorkerPoolXT) getResult(job Job) {
	select {
	case response := <-job.response:
		wp.responsesChan <- response

	case <-job.ctx.Done():
		switch job.ctx.Err() {
		case context.DeadlineExceeded:
			wp.responsesChan <- newResponseError(context.DeadlineExceeded, job.Name, job.startTime)

		case context.Canceled:
			wp.responsesChan <- newResponseError(context.Canceled, job.Name, job.startTime)
		}
	}
}

// newPayload converts our job into a func based upon all options we were given
func (wp *WorkerPoolXT) newPayload(job Job) func() error {
	job.backoff = wp.getBackoff(job)

	return func() error {
		o := wp.getOptions(job)
		r := job.Task(o)

		// If Job not using Retry, don't want to return an error
		if r.Error != nil && job.backoff != nil {
			return r.Error
		}

		r.runtimeDuration = time.Since(job.startTime)
		r.name = job.Name

		if job.ctx.Err() == nil {
			job.response <- r
		}

		// We can return nil whether Job is using Retry or not
		return nil
	}
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

// stop either stops the worker pool now or later
func (wp *WorkerPoolXT) stop(now bool) {
	wp.once.Do(func() {
		if now {
			wp.Stop()
		} else {
			wp.StopWait()
		}
		close(wp.responsesChan)
		wp.killswitch <- struct{}{}
	})
}

// wrap generates the func that we pass to Submit.
func (wp *WorkerPoolXT) wrap(job Job) func() {
	ctx, done := wp.getContext(job)

	return func() {
		job.metadata = &metadata{
			ctx:       ctx,
			done:      done,
			response:  make(chan Response),
			startTime: time.Now(),
		}

		go wp.do(job)
		wp.getResult(job)
	}
}

// Options hold misc options
type Options map[string]interface{}

// Job holds job data
type Job struct {
	*metadata
	Name    string
	Task    func(Options) Response
	Context context.Context
	Options Options
	Retry   int
}

type metadata struct {
	backoff   backoff.BackOff
	ctx       context.Context
	done      context.CancelFunc
	response  chan Response
	startTime time.Time
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

// helper func
func newResponseError(err error, jobName string, start time.Time) Response {
	return Response{
		Error:           err,
		name:            jobName,
		runtimeDuration: time.Since(start),
	}
}

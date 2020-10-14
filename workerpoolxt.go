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
//
// TODO... I would love a 'native' way to retry jobs, though!
//
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

func (wp *WorkerPoolXT) newJobTask(c context.Context, d context.CancelFunc, j Job, t time.Time, b backoff.BackOff) func() error {
	return func() error {
		o := wp.getOptions(j)
		f := j.Task(o)

		// We only want to return an error if the job is using Retry
		// because we are using `backoff`. For the retry to work
		// `backoff` has to know the job failed, this is how we tell it.
		// If the job isn't using Retry/backoff, returning an error
		// will cause problems.
		if f.Error != nil && b != nil {
			return f.Error
		}

		f.runtimeDuration = time.Since(t)
		f.name = j.Name

		// This check is important as it keeps from sending
		// duplicate responses on our responses chan
		// ** See `./doc.go`, section `Response Channel`
		// for more info **
		if c.Err() == nil {
			wp.responsesChan <- f
		}

		// This needs to be here in case the job is using Retry
		// If the job is NOT using Retry, it is OK to return nil
		// here. It will not cause issues like returning an error
		// would.
		return nil
	}
}

// work should be ran on it's own goroutine. Sorts out job metadata and options and runs user provided Job.Task
func (wp *WorkerPoolXT) work(ctx context.Context, done context.CancelFunc, j Job, ts time.Time, bo backoff.BackOff) {
	jobtask := wp.newJobTask(ctx, done, j, ts, bo)

	// Set the default "todo" func
	todo := func() {
		jobtask()
	}

	// If the job is using Retry, we need to use `backoff`
	// to control the retries so we have to handle things
	// differently. We can't just call what we are given.
	// We set our  "todo" func so that it handles retries
	// accordingly.
	if bo != nil {
		todo = func() {
			jobErr := backoff.Retry(jobtask, bo)
			if jobErr != nil {
				wp.responsesChan <- Response{
					name:  j.Name,
					Error: jobErr,
				}
			}
		}
	}

	// Call whatever "it" is we need to do
	todo()
	// Cancel our context
	done()
}

// wrap generates the func that we pass to Submit.
func (wp *WorkerPoolXT) wrap(job Job) func() {
	jobctx, jobdone := wp.getContext(job)

	// This is the func() that ultimately gets passed to `workerpool.Submit(f)`
	return func() {
		start := time.Now()
		retryBackoff := wp.getBackoff(job)

		// Wrap the func before passing to workerpool.Submit
		go wp.work(jobctx, jobdone, job, start, retryBackoff)

		select {
		case <-jobctx.Done():
			switch jobctx.Err() {
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

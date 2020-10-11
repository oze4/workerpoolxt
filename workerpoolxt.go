package workerpoolxt

import (
	"context"
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

// WorkerPoolXT extends `github.com/gammazero/workerpool` by:
//  - Collects the output of all jobs so you can work
//    with it later
//  - Job runtime duration stats baked in
//  - Job timeouts; global timeout or per job basis
//  - Lets you "pause" to gather results at any given
//    time
type WorkerPoolXT struct {
	// Embed WorkerPool
	*workerpool.WorkerPool
	// defaultTimeout is the default timeout for each job
	// If a timeout is not specified along with the job,
	// we use this timeout
	defaultTimeout time.Duration
	// responsesChan is a chan to send job results on
	responsesChan chan Response
	// responses stores job results. Since jobs run
	// as soon as they are submitted, we have to store the
	// results so we can return them when `StopWaitXT()`
	// is called
	responses []Response
	// No upper limit on # of jobs queued, only workers
	// active means we canprovide an unbuffered responsesChan
	// without worrying about deadlock but we also need to
	// know when to stop blocking. When anything is sent to
	// the killswitch chan we stop processing Responses
	killswitch chan bool
	// options are optional default options that get passed to each job
	options Options
}

// SubmitXT submits a job. Allows you to not only
// submit a job, but get the response from it
func (wp *WorkerPoolXT) SubmitXT(job Job) {
	wp.Submit(wp.wrap(job))
}

// StopWaitXT gets results then kills the worker pool
// - You cannot add jobs after calling `StopWaitXT()``
// - Wrapper for `workerpool.StopWait()`
func (wp *WorkerPoolXT) StopWaitXT() (rs []Response) {
	wp.StopWait()
	wp.stop()
	return wp.responses
}

// WithOptions sets default options for each job
// You can also supply options on a per job basis,
// which will override the default options.
func (wp *WorkerPoolXT) WithOptions(o Options) {
	wp.options = o
}

// wrap generates the func that we pass to Submit.
// - If a timeout is not supplied with the job, we use
//   the global default supplied when `New()` is called
// - If optoins are not supplied with the job, we use
//   the global default supplied when `New()` is called
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
			// This check is important as it keeps from sending
			// duplicate responses on our responses chan
			if ctx.Err() == nil {
				wp.responsesChan <- f
			}
			done()
		}(ctx, cancel, job, start)

		select {
		// If our timeout has passed, return an error object,
		// disregarding any response from job which timed out
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
	if wp.options == nil && job.Options == nil {
		return make(Options)
	}
	if wp.options != nil {
		return wp.options
	}
	return job.Options
}

// processResponses listens for anything on the responses chan and appends
// the response to wp.responses - No upper limit on # of jobs queued, only
// workers active means we can provide an unbuffered responsesChan without
// worrying about deadlock
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

// kill sends the signal to stop blocking and return
func (wp *WorkerPoolXT) kill() {
	wp.killswitch <- true
}

// stop closes response chan and triggers our killswitch
func (wp *WorkerPoolXT) stop() {
	close(wp.responsesChan)
	wp.kill()
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

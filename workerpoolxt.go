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
//  - Collects the output of all jobs so you can work with it later
//  - Job runtime duration stats baked in
//  - Job timeouts; global timeout or per job basis
//  - Lets you "pause" to gather results at any given time
type WorkerPoolXT struct {
	*workerpool.WorkerPool
	// Job timeout
	// Default global job timeout. When a timeout is not specified
	// along with the job, we use this timeout
	defaultTimeout time.Duration
	// Chan to send job results on
	responsesChan chan Response
	// Store job results
	// Since jobs run as soon as they are submitted, we have to store them
	// so we can return them when `StopWaitXT()` is called
	responses []Response
	// No upper limit on # of jobs queued, only workers active means we can
	// provide an unbuffered responsesChan without worrying about deadlock but
	// we also need to know when to stop blocking. When anything is sent to
	// the killswitch chan we stop processing Responses
	killswitch chan bool
}

// SubmitXT submits a job
// Allows you to not only submit a job, but get the response from it
func (r *WorkerPoolXT) SubmitXT(job Job) {
	r.Submit(r.wrap(job))
}

// StopWaitXT gets results then kills the worker pool
// - You cannot add jobs after calling `StopWaitXT()``
// - Wrapper for `workerpool.StopWait()`
func (r *WorkerPoolXT) StopWaitXT() (rs []Response) {
	r.StopWait()
	r.stop()
	return r.responses
}

// wrap generates the func that we pass to Submit.
// - If a timeout is not supplied with the job, we use the global default supplied when `New()` is called
// - Responsible for injecting timeout and runtime duration
func (r *WorkerPoolXT) wrap(job Job) func() {
	timeout := r.defaultTimeout
	if job.Timeout != 0 {
		timeout = job.Timeout
	}

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		start := time.Now()

		go func(ctx context.Context, done context.CancelFunc, jj Job, timerStart time.Time) {
			j := jj.Task()
			j.runtimeDuration = time.Since(timerStart)
			j.name = jj.Name
			// This check is important as it keeps from sending duplicate responses on our responses chan
			if ctx.Err() == nil {
				r.responsesChan <- j
			}
			done()
		}(ctx, cancel, job, start)

		select {
		case <-ctx.Done():
			switch ctx.Err() {
			case context.DeadlineExceeded:
				r.responsesChan <- Response{
					Error:           context.DeadlineExceeded,
					name:            job.Name,
					runtimeDuration: time.Since(start),
				}
			}
		}
	}
}

// processResponses listens for anything on the responses chan and appends
// the response to r.responses - No upper limit on # of jobs queued, only
// workers active means we can provide an unbuffered responsesChan without
// worrying about deadlock
func (r *WorkerPoolXT) processResponses() {
	for {
		select {
		case response, ok := <-r.responsesChan:
			if !ok {
				goto Done
			}
			r.responses = append(r.responses, response)
		}
	}
Done:
	<-r.killswitch
}

// kill sends the signal to stop blocking and return
func (r *WorkerPoolXT) kill() {
	r.killswitch <- true
}

// stop closes response chan and triggers our killswitch
func (r *WorkerPoolXT) stop() {
	close(r.responsesChan)
	r.kill()
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

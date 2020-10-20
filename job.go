package workerpoolxt

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
)

// Job holds job data
type Job struct {
	Name      string
	Task      func(Options) Result
	Context   context.Context
	Options   Options
	Retry     int
	childCtx  context.Context    // childCtx is "child" context of Job.Context, lets us "catch" parent Context.Err()
	done      context.CancelFunc // done is the cancelFunc for childCtx
	result    chan Result        // result is the chan we send job reslts on
	startedAt time.Time          // startedAt is the time at which the job started
}

// Options hold misc options
type Options map[string]interface{}

// payload is a `func() error`
type payload func() error

func (p payload) toBackOffOperation() backoff.Operation {
	return (backoff.Operation)(p)
}

// errResult returns a new result based upon your error
func (j *Job) errResult(err error) Result {
	return Result{
		Error:    err,
		duration: time.Since(j.startedAt),
	}
}

// getResult listens for something on the result chan as well
// as for any child ctx errors, whichever happens first
func (j *Job) getResult() Result {
	select {
	case r := <-j.result:
		return r
	case <-j.childCtx.Done():
		switch j.childCtx.Err() {
		default:
			return j.errResult(j.childCtx.Err())
		}
	}
}

// run calls Job.Task using provided variables accordingly
func (j *Job) run() {
	payload := j.toPayload()

	// Job not using retry, just call the payload, no special handling needed
	f := func() {
		payload()
	}

	// Job using retry, wrap our payload with backoff before calling
	if j.Retry > 0 {
		b := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), uint64(j.Retry))
		f = func() {
			err := backoff.Retry(payload.toBackOffOperation(), b)
			if err != nil {
				// Since our payload will be sending the success result (if there is one)
				// we only need to handle job errors that backoff gives us
				j.result <- j.errResult(err)
			}
		}
	}

	f()
}

// runDone runs the job and calls done (which is a context.cancelFunc)
func (j *Job) runDone() {
	j.run()
	j.done()
}

// toPayload converts our job into the correct type so we can use package `backoff` for retry purposes
func (j *Job) toPayload() payload {
	// Our payload is crafted differently if a Job is using Retry
	return func() error {
		r := j.Task(j.Options)
		r.duration = time.Since(j.startedAt)
		r.name = j.Name

		// Only return error if job is using retry
		if r.Error != nil && j.Retry > 0 {
			return r.Error
		}

		// Send our result to our result chan
		j.result <- r

		// Unlike returning an error it does not matter if we return nil here
		return nil
	}
}

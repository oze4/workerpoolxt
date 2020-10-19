package workerpoolxt

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
)

// Options hold misc options
type Options map[string]interface{}

// metadata holds misc data, etc... about each job
type metadata struct {
	ctx       context.Context
	done      context.CancelFunc
	result    chan Result
	startedAt time.Time
}

// payload is a `func() error`
type payload func() error

func (p payload) toBackOffOperation() backoff.Operation {
	return (backoff.Operation)(p)
}

// Job holds job data
type Job struct {
	*metadata
	Name    string
	Task    func(Options) Result
	Context context.Context
	Options Options
	Retry   int
}

// errResult returns a new result based upon your error
func (j *Job) errResult(err error) Result {
	return Result{
		Error:    err,
		duration: time.Since(j.startedAt),
	}
}

// toFunc converts our job into a `func()` so we can call it later
// fallback options are equal to "default" options and will be used if
// options are not provided with the Job (job options override default options)
func (j *Job) toFunc(fallback Options) func() {
	payload := j.toPayload(fallback)

	// Job not using retry
	f := func() {
		payload()
	}

	// Job using retry, wrap our payload with backoff before calling
	if j.Retry > 0 {
		b := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), uint64(j.Retry))
		f = func() {
			err := backoff.Retry(payload.toBackOffOperation(), b)
			if err != nil {
				j.result <- j.errResult(err)
			}
		}
	}

	return f
}

func (j *Job) toPayload(fallback Options) payload {
	return func() error {
		opts := fallback
		if j.Options != nil {
			opts = j.Options
		}

		r := j.Task(opts)
		// Only return error if job is using retry
		if r.Error != nil && j.Retry > 0 {
			return r.Error
		}

		r.duration = time.Since(j.startedAt)
		r.name = j.Name

		j.result <- r
		// Return nil whether job is using retry or not
		return nil
	}
}

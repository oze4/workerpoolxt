package workerpoolxt

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
)

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

// withBackoff wraps our payload. Only need to do this because the job may be using retry.
// otherwise we would just call the payload without checking
func (j *Job) withRetry(payload func() error) func() {
	// Job using retry, using backoff package to handle retries
	return func() {
		err := backoff.Retry(payload, j.bo)
		if err != nil {
			j.result <- j.errResult(err)
		}
	}
}

// metadata is mostly for organizational purposes. Holds misc data, etc... about each job
type metadata struct {
	bo        backoff.BackOff
	ctx       context.Context
	done      context.CancelFunc
	result    chan Result
	startedAt time.Time
}

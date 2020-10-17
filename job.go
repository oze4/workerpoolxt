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

// errResultCtx creates a new result using the current ctx error
func (j *Job) errResultCtx() Result {
	return j.errResult(j.ctx.Err())
}

// metadata is mostly for organizational purposes. Holds misc data, etc... about each job
type metadata struct {
	bo        backoff.BackOff
	ctx       context.Context
	done      context.CancelFunc
	result    chan Result
	startedAt time.Time
}

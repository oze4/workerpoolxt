package workerpoolxt

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
)

// Job holds job data
type Job struct {
	*jobMetadata
	Name    string
	Task    func(Options) Response
	Context context.Context
	Options Options
	Retry   int
}

// errResponse returns a new response based upon your error
func (j *Job) errResponse(err error) Response {
	return Response{
		Error:    err,
		duration: time.Since(j.startedAt),
	}
}

// errResponseCtx creates a new response using the current ctx error
func (j *Job) errResponseCtx() Response {
	return j.errResponse(j.ctx.Err())
}

// jobMetadata is mostly for organizational purposes. Holds misc data, etc... about each job
type jobMetadata struct {
	backoff   backoff.BackOff
	ctx       context.Context
	done      context.CancelFunc
	response  chan Response
	startedAt time.Time
}

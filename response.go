package workerpoolxt

import (
	"time"
)

// Response holds job results
type Response struct {
	Error    error
	Data     interface{}
	name     string
	duration time.Duration
}

// RuntimeDuration returns the amount of time it took to run the job
func (r *Response) RuntimeDuration() time.Duration {
	return r.duration
}

// Name returns the job name
func (r *Response) Name() string {
	return r.name
}

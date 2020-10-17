package workerpoolxt

import (
	"time"
)

// Result holds job results
type Result struct {
	Error    error
	Data     interface{}
	name     string
	duration time.Duration
}

// Duration returns the amount of time it took to run the job
func (r *Result) Duration() time.Duration {
	return r.duration
}

// Name returns the job name
func (r *Result) Name() string {
	return r.name
}

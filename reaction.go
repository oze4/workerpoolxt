package wpreactor

import (
	"time"
)

// Reaction holds response data
type Reaction struct {
	Response interface{}
	Error    error
	duration time.Duration
	name     string
}

// Duration returns duration
func (r *Reaction) Duration() time.Duration {
	return r.duration
}

// Name returns the job name
func (r *Reaction) Name() string {
	return r.name
}

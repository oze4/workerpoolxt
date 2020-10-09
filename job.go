package workerpoolxt

import (
	"time"
)

// Job holds job data
type Job struct {
	Name    string
	Task    func() Response
	Timeout time.Duration
}


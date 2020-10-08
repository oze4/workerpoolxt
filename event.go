package reactor

import (
	"time"
)

// Event holds job data
type Event struct {
	Timeout time.Duration
	Name    string
	Action  Action
}

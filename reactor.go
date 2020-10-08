package reactor

import (
	"context"
	"time"

	"github.com/gammazero/workerpool"
)

type reactor struct {
	jobCount       int
	jobResultCount int
	jobTimeout     time.Duration
	workerPool     *workerpool.WorkerPool
	reactions      chan Reaction
	transport      *Client
}

// React submits a job
func (r *reactor) React(single Event) {
	r.workerPool.Submit(r.wrapper(single))
}

// Reacts allows you to supply multiple Events
func (r *reactor) Reacts(many Events) {
	for _, e := range many {
		r.workerPool.Submit(r.wrapper(e))
	}
}

// ReactionsStop gets results then kills the worker pool
//
// You cannot React (add events to worker pool) after calling `ReactionsStop()``
func (r *reactor) ReactionsStop() Reactions {
	r.workerPool.StopWait()
	close(r.reactions)

	var reactions Reactions
	for reaction := range r.reactions {
		reactions = append(reactions, reaction)
	}

	return reactions
}

// ReactionsWait **SHOULD NOT BE USED YET** I still need to test and make sure
// this is "concurrency-safe".
//
// essentially "pauses" the workerpool to get all current
// and pending event reactions. Once we have all reactions we return them
// and you can continue to use the workerpool.
//
// Unlike `ReactionsStop()` this does not kill the worker pool. You can continue
// to add jobs (events) after calling `ReactionsWait()`
func (r *reactor) ReactionsWait() Reactions {
	var reactions Reactions

	for {
		select {
		case reaction := <-r.reactions:
			reactions = append(reactions, reaction)
			r.jobResultCount++
			if (r.jobCount) == (r.jobResultCount) {
				goto Return
			}
		}
	}

Return:
	return reactions
}

// worker kicks off job and places result on results chan unless timeout is exceeded. If that is
// the case we do nothing with return and let `wrapper` handle context deadline exceeded
func (r *reactor) worker(ctx context.Context, done context.CancelFunc, event Event, start time.Time) {
	reaction := event.Action(r.transport)
	reaction.duration = time.Since(start)
	reaction.name = event.Name

	if ctx.Err() == nil {
		r.reactions <- reaction
	}

	done()
}

// wrapper should be private
func (r *reactor) wrapper(event Event) func() {
	timeout := r.jobTimeout
	// If Event contains a Timeout use it, otherwise use the general timeout
	if event.Timeout != 0 {
		timeout = event.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	r.jobCount = r.jobCount + 1

	return func() {
		start := time.Now()
		go r.worker(ctx, cancel, event, start)

		select {
		case <-ctx.Done():
			switch ctx.Err() {
			case context.DeadlineExceeded:
				r.reactions <- Reaction{
					Error:    context.DeadlineExceeded,
					name:     event.Name,
					duration: time.Since(start),
				}
			}
		}
	}
}

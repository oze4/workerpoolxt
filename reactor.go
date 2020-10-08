package reactor

import (
	"context"
	"net/http"
	"time"

	"github.com/gammazero/workerpool"
	"k8s.io/client-go/kubernetes"
)

// New creates a new Reactor
func New(maxWorkers int, jobTimeout time.Duration) Reactor {
	// Do whatever you need to here to create default client
	defaultClient := &Client{
		HTTP:       http.Client{},
		Kubernetes: kubernetes.Clientset{},
	}

	return &reactor{
		workerPool: workerpool.New(maxWorkers),
		jobTimeout: jobTimeout,
		transport:  defaultClient,
		reactions:  make(chan Reaction, 100),
	}
}

// NewWithClient creates a new Reactor with a custom client
func NewWithClient(client *Client, maxWorkers int, jobTimeout time.Duration) Reactor {
	return &reactor{
		workerPool: workerpool.New(maxWorkers),
		jobTimeout: jobTimeout,
		transport:  client,
		reactions:  make(chan Reaction, 100),
	}
}

// NewEvent creates a new event
func NewEvent(name string, action Action) Event {
	return Event{
		Name:   name,
		Action: action,
	}
}

// Reactions holds reactions
type Reactions []Reaction

// Events holds multiple Event
type Events []Event

// Action is a func that takes a *Client and returns a Reaction
type Action func(*Client) Reaction

// Reactor knows how to handle jobs
type Reactor interface {
	// React puts a job on the queue
	React(single Event)
	// Reacts allows you to supply multiple Events
	Reacts(many Events)
	// Stop workerpool and get results
	// **CANNOT** call React(...) or Reacts(...) after calling ReactionStop()
	ReactionsStop() Reactions
	// Wait on results without stopping worker pool, then continue
	// **CAN** continue to call React(...) or Reacts(...) after calling ReactionWait()
	ReactionsWait() Reactions
}

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
	ctx, cancel := context.WithTimeout(context.Background(), r.jobTimeout)
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

// Client holds http and kubernetes clients
type Client struct {
	HTTP       http.Client
	Kubernetes kubernetes.Clientset
}

// Event holds job data
type Event struct {
	Name   string
	Action Action
}

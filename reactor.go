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

// Reactions holds reactions
type Reactions []Reaction

// Reactor knows how to handle jobs
type Reactor interface {
	React(to Event) // React puts a job on the queue
	Get() Reactions // Get results
}

type reactor struct {
	jobTimeout time.Duration
	workerPool *workerpool.WorkerPool
	reactions  chan Reaction
	transport  *Client
}

// React submits a job
func (r *reactor) React(j Event) {
	r.workerPool.Submit(r.wrapper(j))
}

// Get gets results
// Only call after you have finished adding jobs
func (r *reactor) Get() Reactions {
	return r.getResults()
}

func (r *reactor) getResults() Reactions {
	r.workerPool.StopWait()
	close(r.reactions)

	var allReacts []Reaction
	for jobreact := range r.reactions {
		allReacts = append(allReacts, jobreact)
	}

	return allReacts
}

// worker should be private
func (r *reactor) worker(ctx context.Context, done context.CancelFunc, job Event, start time.Time) {
	reaction := job.Action(r.transport)
	reaction.duration = time.Since(start)
	reaction.name = job.Name

	if ctx.Err() == nil {
		r.reactions <- reaction
	}

	done()
}

// wrapper should be private
func (r *reactor) wrapper(job Event) func() {
	ctx, cancel := context.WithTimeout(context.Background(), r.jobTimeout)

	return func() {
		start := time.Now()
		go r.worker(ctx, cancel, job, start)

		select {
		case <-ctx.Done():
			switch ctx.Err() {
			case context.DeadlineExceeded:
				r.reactions <- Reaction{
					Error:    context.DeadlineExceeded,
					name:     job.Name,
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
	Action func(*Client) Reaction
}

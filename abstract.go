package reactor

import (
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

// Reactions holds reactions
type Reactions []Reaction

// Events holds multiple Event
type Events []Event

// Action is a func that takes a *Client and returns a Reaction
type Action func(*Client) Reaction

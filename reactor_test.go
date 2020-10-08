package reactor

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gammazero/workerpool"
)

var (
	maxworkers = 3
	jobtimeout = time.Duration(time.Second * 4)
)

func TestNew(t *testing.T) {
	type args struct {
		maxWorkers int
		jobTimeout time.Duration
	}
	tests := []struct {
		name string
		args args
		want Reactor
	}{
		{
			args: args{
				maxWorkers: maxworkers,
				jobTimeout: jobtimeout,
			},
			want: New(maxworkers, jobtimeout),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.maxWorkers, tt.args.jobTimeout); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewWithClient(t *testing.T) {
	type args struct {
		client     *Client
		maxWorkers int
		jobTimeout time.Duration
	}
	tests := []struct {
		name string
		args args
		want Reactor
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewWithClient(tt.args.client, tt.args.maxWorkers, tt.args.jobTimeout); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewWithClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewEvent(t *testing.T) {
	type args struct {
		name   string
		action Action
	}
	tests := []struct {
		name string
		args args
		want Event
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewEvent(tt.args.name, tt.args.action); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_reactor_React(t *testing.T) {
	type fields struct {
		jobTimeout     time.Duration
		workerPool     *workerpool.WorkerPool
		reactions      chan Reaction
		transport      *Client
		jobCount       int
		jobResultCount int
	}
	type args struct {
		single Event
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &reactor{
				jobTimeout:     tt.fields.jobTimeout,
				workerPool:     tt.fields.workerPool,
				reactions:      tt.fields.reactions,
				transport:      tt.fields.transport,
				jobCount:       tt.fields.jobCount,
				jobResultCount: tt.fields.jobResultCount,
			}
			r.React(tt.args.single)
		})
	}
}

func Test_reactor_Reacts(t *testing.T) {
	type fields struct {
		jobTimeout     time.Duration
		workerPool     *workerpool.WorkerPool
		reactions      chan Reaction
		transport      *Client
		jobCount       int
		jobResultCount int
	}
	type args struct {
		many Events
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &reactor{
				jobTimeout:     tt.fields.jobTimeout,
				workerPool:     tt.fields.workerPool,
				reactions:      tt.fields.reactions,
				transport:      tt.fields.transport,
				jobCount:       tt.fields.jobCount,
				jobResultCount: tt.fields.jobResultCount,
			}
			r.Reacts(tt.args.many)
		})
	}
}

func Test_reactor_ReactionsStop(t *testing.T) {
	type fields struct {
		jobTimeout     time.Duration
		workerPool     *workerpool.WorkerPool
		reactions      chan Reaction
		transport      *Client
		jobCount       int
		jobResultCount int
	}
	tests := []struct {
		name   string
		fields fields
		want   Reactions
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &reactor{
				jobTimeout:     tt.fields.jobTimeout,
				workerPool:     tt.fields.workerPool,
				reactions:      tt.fields.reactions,
				transport:      tt.fields.transport,
				jobCount:       tt.fields.jobCount,
				jobResultCount: tt.fields.jobResultCount,
			}
			if got := r.ReactionsStop(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reactor.ReactionsStop() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_reactor_ReactionsWait(t *testing.T) {
	type fields struct {
		jobTimeout     time.Duration
		workerPool     *workerpool.WorkerPool
		reactions      chan Reaction
		transport      *Client
		jobCount       int
		jobResultCount int
	}
	tests := []struct {
		name   string
		fields fields
		want   Reactions
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &reactor{
				jobTimeout:     tt.fields.jobTimeout,
				workerPool:     tt.fields.workerPool,
				reactions:      tt.fields.reactions,
				transport:      tt.fields.transport,
				jobCount:       tt.fields.jobCount,
				jobResultCount: tt.fields.jobResultCount,
			}
			if got := r.ReactionsWait(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reactor.ReactionsWait() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_reactor_worker(t *testing.T) {
	type fields struct {
		jobTimeout     time.Duration
		workerPool     *workerpool.WorkerPool
		reactions      chan Reaction
		transport      *Client
		jobCount       int
		jobResultCount int
	}
	type args struct {
		ctx   context.Context
		done  context.CancelFunc
		event Event
		start time.Time
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &reactor{
				jobTimeout:     tt.fields.jobTimeout,
				workerPool:     tt.fields.workerPool,
				reactions:      tt.fields.reactions,
				transport:      tt.fields.transport,
				jobCount:       tt.fields.jobCount,
				jobResultCount: tt.fields.jobResultCount,
			}
			r.worker(tt.args.ctx, tt.args.done, tt.args.event, tt.args.start)
		})
	}
}

/*func Test_reactor_wrapper(t *testing.T) {
	type fields struct {
		jobTimeout     time.Duration
		workerPool     *workerpool.WorkerPool
		reactions      chan Reaction
		transport      *Client
		jobCount       int
		jobResultCount int
	}
	type args struct {
		event Event
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   func()
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &reactor{
				jobTimeout:     tt.fields.jobTimeout,
				workerPool:     tt.fields.workerPool,
				reactions:      tt.fields.reactions,
				transport:      tt.fields.transport,
				jobCount:       tt.fields.jobCount,
				jobResultCount: tt.fields.jobResultCount,
			}
			if got := r.wrapper(tt.args.event); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reactor.wrapper() = %v, want %v", got, tt.want)
			}
		})
	}
}*/

func TestReaction_Duration(t *testing.T) {
	type fields struct {
		Response interface{}
		Error    error
		duration time.Duration
		name     string
	}
	tests := []struct {
		name   string
		fields fields
		want   time.Duration
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reaction{
				Response: tt.fields.Response,
				Error:    tt.fields.Error,
				duration: tt.fields.duration,
				name:     tt.fields.name,
			}
			if got := r.Duration(); got != tt.want {
				t.Errorf("Reaction.Duration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReaction_Name(t *testing.T) {
	type fields struct {
		Response interface{}
		Error    error
		duration time.Duration
		name     string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reaction{
				Response: tt.fields.Response,
				Error:    tt.fields.Error,
				duration: tt.fields.duration,
				name:     tt.fields.name,
			}
			if got := r.Name(); got != tt.want {
				t.Errorf("Reaction.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

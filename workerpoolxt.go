package workerpoolxt

import (
	"context"
	"sync"
	"time"

	"github.com/gammazero/workerpool"
)

// New creates WorkerPoolXT
func New(ctx context.Context, maxWorkers int) *WorkerPoolXT {
	p := &WorkerPoolXT{
		WorkerPool: workerpool.New(maxWorkers),
		context:    ctx,
		result:     make(chan Result),
		kill:       make(chan struct{}),
	}
	go p.processResults()
	return p
}

// NewWithOptions creates a new WorkerPool with options
func NewWithOptions(ctx context.Context, maxWorkers int, o Options) *WorkerPoolXT {
	p := New(ctx, maxWorkers)
	p.options = o
	return p
}

// WorkerPoolXT extends `github.com/gammazero/workerpool`
type WorkerPoolXT struct {
	*workerpool.WorkerPool
	context context.Context
	kill    chan struct{}
	options Options
	once    sync.Once
	result  chan Result
	results []Result
}

// SubmitXT submits a job which you can get a result from
func (p *WorkerPoolXT) SubmitXT(j Job) {
	p.Submit(p.wrap(&j))
}

// StopWaitXT gets results then kills the worker pool
func (p *WorkerPoolXT) StopWaitXT() (rs []Result) {
	p.stop(false)
	return p.results
}

// do converts our job into a Payload and does it
func (p *WorkerPoolXT) do(j *Job) {
	// Convert job to func and immediately invoke it
	j.toFunc(p.options)()
	j.done()
}

// getContext decides which context to use : default or job
func (p *WorkerPoolXT) getContext(j *Job) (context.Context, context.CancelFunc) {
	if j.Context != nil {
		return context.WithCancel(j.Context)
	}
	return context.WithCancel(p.context)
}

// getResults sends job results to a channel
func (p *WorkerPoolXT) getResult(j *Job) {
	select {
	case r := <-j.result:
		p.result <- r
	case <-j.ctx.Done():
		switch j.ctx.Err() {
		default:
			p.result <- j.errResult(j.ctx.Err())
		}
	}
}

// processResults listens for results on resultsChan
func (p *WorkerPoolXT) processResults() {
	for {
		select {
		case result, ok := <-p.result:
			if !ok {
				goto Done
			}
			p.results = append(p.results, result)
		}
	}
Done:
	<-p.kill
}

// stop either stops the worker pool now or later
func (p *WorkerPoolXT) stop(now bool) {
	p.once.Do(func() {
		if now {
			p.Stop()
		} else {
			p.StopWait()
		}
		close(p.result)
		p.kill <- struct{}{}
	})
}

// wrap generates the func that we pass to Submit.
func (p *WorkerPoolXT) wrap(j *Job) func() {
	ctx, done := p.getContext(j)
	// This is the func we ultimately pass to `workerpool`
	return func() {
		j.metadata = &metadata{
			ctx:       ctx,
			done:      done,
			result:    make(chan Result),
			startedAt: time.Now(),
		}

		go p.do(j)
		p.getResult(j)
	}
}

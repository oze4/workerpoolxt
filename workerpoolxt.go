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
	// This is the func we ultimately pass to `workerpool`
	return func() {
		// Allow job options to override default pool options
		if j.Options == nil {
			j.Options = p.options
		}

		if j.Context == nil {
			j.Context = p.context
		}

		j.childCtx, j.done = context.WithCancel(j.Context)
		j.result = make(chan Result)
		j.startedAt = time.Now()

		go j.runDone()
		p.result <- j.getResult()
	}
}

package workerpoolxt

import (
	"context"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
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
	p.Submit(p.wrap(j))
}

// StopWaitXT gets results then kills the worker pool
func (p *WorkerPoolXT) StopWaitXT() (rs []Result) {
	p.stop(false)
	return p.results
}

// WithOptions sets default options for each job
func (p *WorkerPoolXT) WithOptions(o Options) {
	p.options = o
}

// do gets the appropriate payload and does it
func (p *WorkerPoolXT) do(j Job) {
	j.bo = p.getBackoff(j)
	// Need to change how we call our payload if job is using retry
	payload := p.newPayload(j)
	// Job not using retry, just call what we were given
	if j.bo == nil {
		payload()
	} else {
		// Job using retry, wrap our payload with backoff before calling
		j.withRetry(payload)()
	}
	j.done()
}

// getBackoff determines if a job is using Retry
// TODO I would love a 'native' way to retry jobs, though!
func (p *WorkerPoolXT) getBackoff(j Job) backoff.BackOff {
	if j.Retry > 0 {
		return backoff.WithMaxRetries(backoff.NewExponentialBackOff(), uint64(j.Retry))
	}
	return nil
}

// getContext decides which context to use : default or job
func (p *WorkerPoolXT) getContext(j Job) (context.Context, context.CancelFunc) {
	if j.Context != nil {
		return context.WithCancel(j.Context)
	}
	return context.WithCancel(p.context)
}

// getOptions decides which options to use : default or job
func (p *WorkerPoolXT) getOptions(j Job) Options {
	// job options should override default pool options
	if j.Options != nil {
		return j.Options
	}
	if p.options != nil {
		return p.options
	}
	return nil
}

// getResult reacts to a jobs context
func (p *WorkerPoolXT) getResult(j Job) {
	select {
	case r := <-j.result: // Have job result, feed to pool
		p.result <- r
	case <-j.ctx.Done():
		switch j.ctx.Err() { // "catch" any context errors
		default:
			p.result <- j.errResult(j.ctx.Err())
		}
	}
}

// newPayload converts our job into a func based upon all options we were given
func (p *WorkerPoolXT) newPayload(j Job) func() error {
	// If job using retry, we need to create our payload differently
	return func() error {
		o := p.getOptions(j)
		r := j.Task(o)
		// Only return error if job is using retry
		if r.Error != nil && j.bo != nil {
			return r.Error
		}
		r.duration = time.Since(j.startedAt)
		r.name = j.Name
		j.result <- r
		// Return nil whether job is using retry or not
		return nil
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
func (p *WorkerPoolXT) wrap(j Job) func() {
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

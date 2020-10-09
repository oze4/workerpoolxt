package workerpoolxt

import (
	//"reflect"
	"testing"
	"time"
)

func TestOverflow(t *testing.T) {
	wp := New(2, time.Duration(time.Second*10))
	releaseChan := make(chan struct{})

	// Start workers, and have them all wait on a channel before completing.
	for i := 0; i < 64; i++ {
		wp.SubmitXT(Job{
			Name: "test1",
			Task: func() Response {
				<-releaseChan
				return Response{}
			},
		})
	}

	// Start a goroutine to free the workers after calling stop.  This way
	// the dispatcher can exit, then when this goroutine runs, the workerpool
	// can exit.
	go func() {
		<-time.After(time.Millisecond)
		close(releaseChan)
	}()
	wp.Stop()

	// Now that the worker pool has exited, it is safe to inspect its waiting
	// queue without causing a race.
	qlen := wp.WorkerPool.WaitingQueueSize()
	if qlen != 62 {
		t.Fatal("Expected 62 tasks in waiting queue, have", qlen)
	}
}

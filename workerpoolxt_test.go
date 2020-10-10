package workerpoolxt

import (
	//"reflect"
	"fmt"
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

func TestSubmitXT_HowToHandleErrors(t *testing.T) {
	wp := New(3, time.Duration(time.Second*10))
	wp.SubmitXT(Job{ // Uses default timeout
		Name: "Job 1 will pass",
		Task: func() Response {
			return Response{Data: "yay"}
		}})
	wp.SubmitXT(Job{ // Uses custom timeout
		Name:    "Job 2 will timeout",
		Timeout: time.Duration(time.Millisecond * 1),
		Task: func() Response {
			time.Sleep(time.Second * 20) // Simulate long running task
			return Response{Data: "uhoh"}
		}})
	wp.SubmitXT(Job{ // Or if you encounter an error within the code in your job
		Name: "Job 3 will encounter an error",
		Task: func() Response {
			err := fmt.Errorf("ErrorPretendException : something failed")
			if err != nil {
				return Response{Error: err}
			}
			return Response{Data: "uhoh"}
		}})
	results := wp.StopWaitXT()
	failed, succeeded := 0, 0
	for _, r := range results {
		if r.Error != nil {
			failed++
		} else {
			succeeded++
		}
	}
	if succeeded != 1 || failed != 2 {
		t.Fatalf("expected succeeded=1:failed=2 : got succeeded=%d:failed=%d", succeeded, failed)
	}
}

func TestResultCountEqualsJobCount(t *testing.T) {
	numJobs := 500
	numworkers := 10
	wp := New(numworkers, time.Duration(time.Second*10))
	for i := 0; i < numJobs; i++ {
		ii := i
		wp.SubmitXT(Job{
			Name: fmt.Sprintf("Job %d", ii),
			Task: func() Response { return Response{Data: fmt.Sprintf("Placeholder : %d", ii)} },
		})
	}
	results := wp.StopWaitXT()
	numResults := len(results)
	if numResults != numJobs {
		t.Fatalf("Expected %d results but got %d", numJobs, numResults)
	}
}

func TestRuntimeDuration(t *testing.T) {
	wp := New(3, time.Duration(time.Second*10))
	wp.SubmitXT(Job{
		Name: "test",
		Task: func() Response {
			time.Sleep(time.Second)
			return Response{Data: "testing"}
		},
	})

	res := wp.StopWaitXT()
	first := res[0]
	if first.RuntimeDuration() == 0 {
		t.Fatalf("Expected RuntimeDuration() to not equal 0")
	}
}

func TestName(t *testing.T) {
	thename := "test99"
	wp := New(3, time.Duration(time.Second*10))
	wp.SubmitXT(Job{
		Name: thename,
		Task: func() Response {
			return Response{Data: "testing"}
		},
	})

	res := wp.StopWaitXT()
	first := res[0]
	if first.Name() != thename {
		t.Fatalf("Expected Name() to be %s got %s", thename, first.Name())
	}
}

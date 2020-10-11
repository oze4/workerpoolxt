package workerpoolxt

import (
	//"reflect"
	"context"
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
			Task: func(o Options) Response {
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

func TestStopRace(t *testing.T) {
	wp := New(20, time.Duration(time.Second*10))

	// Start and pause all workers.
	ctx, cancel := context.WithCancel(context.Background())
	wp.Pause(ctx)

	const doneCallers = 5
	stopDone := make(chan struct{}, doneCallers)
	for i := 0; i < doneCallers; i++ {
		go func() {
			wp.Stop()
			stopDone <- struct{}{}
		}()
	}

	select {
	case <-stopDone:
		t.Fatal("Stop should not return in any goroutine")
	default:
	}

	cancel()

	timeout := time.After(time.Second)
	for i := 0; i < doneCallers; i++ {
		select {
		case <-stopDone:
		case <-timeout:
			t.Fatal("timedout waiting for Stop to return")
		}
	}
}

// Run this test with race detector to test that using WaitingQueueSize has no
// race condition
func TestWaitingQueueSizeRace(t *testing.T) {
	const (
		goroutines = 10
		tasks      = 20
		workers    = 5
	)
	wp := New(workers, time.Duration(time.Second*10))
	maxChan := make(chan int)
	for g := 0; g < goroutines; g++ {
		go func() {
			max := 0
			// Submit 100 tasks, checking waiting queue size each time.  Report
			// the maximum queue size seen.
			for i := 0; i < tasks; i++ {
				wp.Submit(func() {
					time.Sleep(time.Microsecond)
				})
				waiting := wp.WaitingQueueSize()
				if waiting > max {
					max = waiting
				}
			}
			maxChan <- max
		}()
	}

	// Find maximum queuesize seen by any goroutine.
	maxMax := 0
	for g := 0; g < goroutines; g++ {
		max := <-maxChan
		if max > maxMax {
			maxMax = max
		}
	}
	if maxMax == 0 {
		t.Error("expected to see waiting queue size > 0")
	}
	if maxMax >= goroutines*tasks {
		t.Error("should not have seen all tasks on waiting queue")
	}
}

func TestSubmitXT_HowToHandleErrors(t *testing.T) {
	wp := New(3, time.Duration(time.Second*10))
	wp.SubmitXT(Job{ // Uses default timeout
		Name: "Job 1 will pass",
		Task: func(o Options) Response {
			return Response{Data: "yay"}
		}})
	wp.SubmitXT(Job{ // Uses custom timeout
		Name:    "Job 2 will timeout",
		Timeout: time.Duration(time.Millisecond * 1),
		Task: func(o Options) Response {
			time.Sleep(time.Second * 20) // Simulate long running task
			return Response{Data: "uhoh"}
		}})
	wp.SubmitXT(Job{ // Or if you encounter an error within the code in your job
		Name: "Job 3 will encounter an error",
		Task: func(o Options) Response {
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
			Task: func(o Options) Response { return Response{Data: fmt.Sprintf("Placeholder : %d", ii)} },
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
		Task: func(o Options) Response {
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
		Task: func(o Options) Response {
			return Response{Data: "testing"}
		},
	})

	res := wp.StopWaitXT()
	first := res[0]
	if first.Name() != thename {
		t.Fatalf("Expected Name() to be %s got %s", thename, first.Name())
	}
}

func TestDefaultOptions(t *testing.T) {
	varname := "myvar"
	varvalue := "myval"
	opts := map[string]interface{}{varname: varvalue}
	fmt.Println(opts[varname])
	wp := New(3, time.Duration(time.Second*10))
	wp.WithOptions(opts)
	wp.SubmitXT(Job{
		Name: "testing default options",
		Task: func(o Options) Response {
			// Set data to our opts myvar
			return Response{Data: o[varname]}
		},
	})

	res := wp.StopWaitXT()
	first := res[0]
	data := first.Data
	if data != varvalue {
		t.Fatalf("Expected option %s to be %s but got %s", varname, varvalue, data)
	}
}

func TestPerJobOptions(t *testing.T) {
	wp := New(3, time.Duration(time.Second*10))
	wp.SubmitXT(Job{
		Name: "job 1",
		Task: func(o Options) Response {
			// Set data to our opts myvar
			return Response{Data: o["var"]}
		},
		Options: map[string]interface{}{"var": "job1value"},
	})
	wp.SubmitXT(Job{
		Name: "job 2",
		Task: func(o Options) Response {
			// Set data to our opts myvar
			return Response{Data: o["var"]}
		},
		Options: map[string]interface{}{"var": "job2value"},
	})

	res := wp.StopWaitXT()
	for _, result := range res {
		if result.Name() == "" {
			t.Fatalf("Expected option %s to be %s but got %s", "var", "not ''", "''")
		}
		if result.Data == "" {
			t.Fatalf("Expected data to not be null on : %s", result.Name())
		}
		if result.Name() == "job 1" {
			if result.Data != "job1value" {
				t.Fatalf("Expected %s option 'var' to be %s but got %s", result.Name(), "job1value", result.Data)
			}
		}
		if result.Name() == "job 2" {
			if result.Data != "job2value" {
				t.Fatalf("Expected %s option 'var' to be %s but got %s", result.Name(), "job2value", result.Data)
			}
		}
	}
}

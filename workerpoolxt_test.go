package workerpoolxt

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

/**
 *
 * Any test ending in `_Special` will have the following command ran against it:
 *
 *   > go test -race -run ^Test.*Special$ -count=2000
 *
 */

var (
	defaultTimeout = time.Duration(time.Second * 10)
	freshCtx       = func() context.Context { return context.Background() }
	defaultWorkers = 10
)

func TestBasics(t *testing.T) {
	wp := New(freshCtx(), defaultWorkers)
	wp.SubmitXT(Job{
		Name: "Basics",
		Task: func(o Options) Response {
			return Response{Data: true}
		},
	})
	wp.SubmitXT(Job{
		Name: "Basics",
		Task: func(o Options) Response {
			return Response{Data: true}
		},
	})
	wp.SubmitXT(Job{
		Name: "Basics",
		Task: func(o Options) Response {
			return Response{Error: errors.New("err")}
		},
	})
	results := wp.StopWaitXT()
	if len(results) != 3 {
		t.Fatalf("Expected 3 results got : %d", len(results))
	}
}

func TestJobContextOverridesDefaultContext(t *testing.T) {
	defaultContext := freshCtx()

	jobContextWeWantToOverrideWith, done := context.WithTimeout(freshCtx(), time.Duration(time.Millisecond))
	defer done()

	wp := New(defaultContext, 10)

	wp.SubmitXT(Job{
		Name: "Using default ctx",
		Task: func(o Options) Response {
			time.Sleep(time.Second)
			return Response{Data: "OK"}
		},
	})

	wp.SubmitXT(Job{
		Name:    "Using custom per job ctx",
		Context: jobContextWeWantToOverrideWith,
		Task: func(o Options) Response {
			time.Sleep(time.Second)
			return Response{Data: "OK"}
		},
	})

	results := wp.StopWaitXT()
	errs, succs, expectedSuccs, expectedErrs := 0, 0, 1, 1

	for _, r := range results {
		if r.Error != nil {
			errs++
		} else {
			succs++
		}
	}
	if errs != expectedErrs || succs != expectedSuccs {
		t.Fatalf("Expected errors=%d:success=%d : got errors=%d:success=%d", expectedErrs, expectedSuccs, errs, succs)
	}
}

func TestTimeouts(t *testing.T) {
	defaultCtx := context.Background()
	numWorkers := 10
	wp := New(defaultCtx, numWorkers)
	timeout := time.Duration(time.Millisecond)

	// don't have to use cancelFunc if you dont want, we call it for you on success
	oneMsTimeoutContext, myCncl := context.WithTimeout(context.Background(), timeout)
	defer myCncl()

	wp.SubmitXT(Job{
		Name:    "my ctx job",
		Context: oneMsTimeoutContext,
		Task: func(o Options) Response {
			// Simulate long running task
			time.Sleep(time.Second * 10)
			return Response{Data: "I could be anything"}
		},
	})

	results := wp.StopWaitXT()
	if len(results) != 1 {
		t.Fatalf("Expected 1 result : got %d", len(results))
	}

	first := results[0]
	if first.Error != context.DeadlineExceeded {
		t.Fatalf("Expected err %s : got err %s", context.DeadlineExceeded, first.Error)
	}
}

func TestCancellingContext(t *testing.T) {
	defaultContext := freshCtx()
	wp := New(defaultContext, 10)

	// won't get a response from this job since we cancel it's context
	ctx, cncl := context.WithCancel(defaultContext)
	wp.SubmitXT(Job{
		Name:    "j1",
		Context: ctx,
		Task: func(o Options) Response {
			time.Sleep(time.Millisecond * 10) // Sleep 10 sec
			return Response{Data: "You shouldn't get a failure or success from me"}
		},
	})

	// we still want to make sure we get a response from this job, though
	wp.SubmitXT(Job{
		Name: "j2",
		Task: func(o Options) Response {
			return Response{Data: "OK"}
		},
	})

	// need to cancel ctx for some reason after doing "something"
	time.Sleep(time.Millisecond * 2)
	cncl()

	results := wp.StopWaitXT()

	errs, succs, expectedSuccs, expectedErrs := 0, 0, 1, 1
	for _, r := range results {
		if r.Error != nil {
			errs++
		} else {
			succs++
		}
	}

	if errs != expectedErrs || succs != expectedSuccs {
		t.Fatalf("Expected errors=%d:success=%d : got errors=%d:success=%d", expectedErrs, expectedSuccs, errs, succs)
	}
}

func TestSubmitWithSubmitXT_UsingStopWaitXT_Special(t *testing.T) {
	var totalResults uint64
	wp := New(freshCtx(), defaultWorkers)
	expectedTotalResults := 2
	wp.Submit(func() {
		time.Sleep(time.Millisecond * 2)
		atomic.AddUint64(&totalResults, 1)
	})
	wp.SubmitXT(Job{
		Name: "From SubmitXT()",
		Task: func(o Options) Response {
			time.Sleep(time.Millisecond * 1)
			atomic.AddUint64(&totalResults, 1)
			return Response{Data: "SubmitXT() after sleep"}
		},
	})
	_ = wp.StopWaitXT()
	if int(atomic.LoadUint64(&totalResults)) != expectedTotalResults {
		t.Fatalf("expect %d results : got %d results", expectedTotalResults, totalResults)
	}
}

func TestSubmitWithSubmitXT_UsingStopWait_Special(t *testing.T) {
	var totalResults uint64
	wp := New(freshCtx(), defaultWorkers)
	expectedTotalResults := 2
	wp.Submit(func() {
		time.Sleep(time.Millisecond * 2)
		atomic.AddUint64(&totalResults, 1)
	})
	wp.SubmitXT(Job{
		Name: "From SubmitXT()",
		Task: func(o Options) Response {
			time.Sleep(time.Millisecond * 1)
			atomic.AddUint64(&totalResults, 1)
			return Response{Data: "SubmitXT() after sleep"}
		},
	})
	wp.StopWait()
	if int(atomic.LoadUint64(&totalResults)) != expectedTotalResults {
		t.Fatalf("expect %d results : got %d results", expectedTotalResults, totalResults)
	}
}

func TestOverflow_Special(t *testing.T) {
	thisTestHasToHaveTwoWorkers := 2
	wp := New(freshCtx(), thisTestHasToHaveTwoWorkers)
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
	max := 20
	wp := New(freshCtx(), max)
	workRelChan := make(chan struct{})

	var started sync.WaitGroup
	started.Add(max)

	// Start workers, and have them all wait on a channel before completing.
	for i := 0; i < max; i++ {
		wp.Submit(func() {
			started.Done()
			<-workRelChan
		})
	}

	started.Wait()

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

	close(workRelChan)

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
	wp := New(freshCtx(), workers)
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
	wp := New(freshCtx(), defaultWorkers)

	wp.SubmitXT(Job{ // Uses default timeout
		Name: "Job 1 will pass",
		Task: func(o Options) Response {
			return Response{Data: "yay"}
		}})

	perJobContext, cncl := context.WithTimeout(freshCtx(), time.Duration(time.Millisecond*1))
	defer cncl()
	wp.SubmitXT(Job{ // Uses custom timeout
		Name:    "Job 2 will timeout",
		Context: perJobContext,
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
	wp := New(freshCtx(), defaultWorkers)
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
	wp := New(freshCtx(), defaultWorkers)
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
	wp := New(freshCtx(), defaultWorkers)
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
	varname, varvalue := "myvar", "myval"
	opts := map[string]interface{}{varname: varvalue}
	wp := New(freshCtx(), defaultWorkers)
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
	wp := New(freshCtx(), defaultWorkers)
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

func TestPerJobOptionsOverrideDefaultOptions(t *testing.T) {
	wp := New(freshCtx(), defaultWorkers)
	// set default options so we can verify they were overwritten by per job options
	opts := map[string]interface{}{"default": "value"}
	wp.WithOptions(opts)
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

/**
 * This is a "cheap" test.  More thorough tests need to be performed on wp.stop(true)
 */
func TestStopNow(t *testing.T) {
	wp := New(freshCtx(), defaultWorkers)
	wp.SubmitXT(Job{
		Name: "job 1",
		Task: func(o Options) Response {
			// Set data to our opts myvar
			return Response{Data: "placeholder"}
		},
	})
	wp.stop(true)
}

func TestRetry(t *testing.T) {
	wp := New(freshCtx(), defaultWorkers)
	expectedError := errors.New("simulating error")
	expectedName := "backoff_test"
	expectedResultsLen := 2
	wp.SubmitXT(Job{
		Name: expectedName,
		Task: func(o Options) Response {
			return Response{Error: expectedError}
		},
		Retry: 3,
	})
	wp.SubmitXT(Job{
		Name: "simulate_success",
		Task: func(o Options) Response {
			return Response{Data: "success"}
		},
	})
	results := wp.StopWaitXT()
	if len(results) != expectedResultsLen {
		t.Fatalf("expected results len of %d, got results len of %d", expectedResultsLen, len(results))
	}
	for _, r := range results {
		if r.Name() == expectedName && r.Error != expectedError {
			t.Fatalf("Expected error %s : got error %s", expectedError, r.Error)
		}
	}
}

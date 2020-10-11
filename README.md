# workerpoolxt
[![Build](https://github.com/oze4/workerpoolxt/workflows/Build/badge.svg?branch=master)](https://github.com/oze4/workerpoolxt/actions)
[![codecov](https://codecov.io/gh/oze4/workerpoolxt/branch/master/graph/badge.svg)](https://codecov.io/gh/oze4/workerpoolxt)
[![Go Report Card](https://goreportcard.com/badge/github.com/oze4/workerpoolxt)](https://goreportcard.com/report/github.com/oze4/workerpoolxt)

Worker pool library that extends [github.com/gammazero/workerpool](https://github.com/gammazero/workerpool).

How we extend `workerpool`:

- [Get results from each job](#basic-example)
  - We collect job results so you can work with them later
  - [How to handle errors](#how-to-handle-errors)
  - [How do I know if a job timed out](#how-to-handle-errors)
- [Pass any variable/data/etc into each job via Options](#options)
  - Pass data from outside of the job without having to worry about closures or generators
  - Set [default options on the workerpool](#supply-default-options)
  - or [on a per job basis](#supply-options-per-job)
  - **If a job has options set, it overrides the defaults \**we do not merge options***\*
- Job runtime statistics
  - Runtime duration stats are baked in
  - Access via `somejobresult.RuntimeDuration() //-> time.Duration`
- [Job timeouts](#basic-example)
  - Fine tune timeouts on a per job basis

### Basic example

```golang
package main

import (
	"fmt"
	"time"
	wpxt "github.com/oze4/workerpoolxt"
)

func main() {
	wp := wpxt.New(10, time.Duration(time.Second*10))

	wp.SubmitXT(Job{ // For demo purposes, this job will timeout
		Name:    "My first job",
		Timeout: time.Duration(time.Second * 1),
		//                     ^^^^^^^^^^^^^^^
		Task: func(o wpxt.Options) wpxt.Response {
			time.Sleep(time.Second * 2)
			//         ^^^^^^^^^^^^^^^
			return wpxt.Response{Data: "Hello"}
		},
	})

	// Submit as many jobs as you would like

	results := wp.StopWaitXT()

	for _, r := range results {
		fmt.Printf("%s took %fms\n", r.Name(), r.RuntimeDuration() * time.Millisecond)
	}
}
```

### How to handle errors

How do I know if a job timed out? How do I handle an error in my job?

```golang
package main

import (
	"fmt"
	"time"

	wpxt "github.com/oze4/workerpoolxt"
)

func main() {
	wp := wpxt.New(3, time.Duration(time.Second*10))

        // Uses default timeout
	wp.SubmitXT(wpxt.Job{ 
		Name: "Job 1 will pass",
		Task: func(o wpxt.Options) wpxt.Response {
			return wpxt.Response{Data: "yay"}
		},
	})

        // Uses custom timeout
	// This job is configured to timeout on purpose
	wp.SubmitXT(wpxt.Job{ 
		Name:    "Job 2 will timeout",
		Timeout: time.Duration(time.Millisecond * 1),
		Task: func(o wpxt.Options) wpxt.Response {
		        // Simulate long running task
			time.Sleep(time.Second * 20) 
			return wpxt.Response{Data: "timedout"}
		},
	})

        // Or if you encounter an error within the code in your job
	wp.SubmitXT(wpxt.Job{ 
		Name: "Job 3 will encounter an error",
		Task: func(o wpxt.Options) wpxt.Response {
			err := fmt.Errorf("ErrorPretendException : something failed")
			if err != nil {
				return wpxt.Response{Error: err}
			}
			return wpxt.Response{Data: "error"}
		},
	})

	results := wp.StopWaitXT()

	for _, r := range results {
		if r.Error != nil {
			fmt.Println(r.Name(), "has failed with error :", r.Error.Error())
		} else {
			fmt.Println(r.Name(), "has passed successfully")
		}
	}
}

// ->
// Job 3 will encounter an error has failed with error : ErrorPretendException : something failed
// Job 1 will pass has passed successfully
// Job 2 will timeout has failed with error : context deadline exceeded
```

### Options

 - Providing options is optional
 - Options are nothing more than `map[string]interface{}` so that you may supply anything you wish. This also simplifies accessing options within a job.
 - You can supply options along with the workerpool, or on a per job basis. 
 - **If a job has options set, it overrides the defaults**
 - **We do not merge options**.

#### Supply default options

```golang
import (
    wpxt "github.com/oze4/workerpoolxt"
    // ...
)

wp := wpxt.New(10, time.Duration(time.Second*10))
myopts := map[string]interface{}{
    "myclient": &http.Client{},
}
wp.WithOptions(myopts)

wp.SubmitXT(wpxt.Job{
    Name: "myjob",
    Task: func(o wpxt.Options) wpxt.Response {
        // access options here
        client := o["myclient"]
    }, 
})
```

#### Supply options per job

```golang
import (
    wpxt "github.com/oze4/workerpoolxt"
    // ...
)

wp := wpxt.New(10, time.Duration(time.Second*10))
myclient := &http.Client{}

wp.SubmitXT(wpxt.Job{
    Name: "myjob",
    Options: map[string]interface{}{
        "http": myclient 
    },
    Task: func(o wpxt.Options) wpxt.Response {
        // access options here
        httpclient := o["http"]
    }, 
})
```

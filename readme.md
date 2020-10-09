# workerpoolxt

Worker pool library that extends [github.com/gammazero/workerpool](https://github.com/gammazero/workerpool).

How we extend `workerpool`:

- Get results from each job
  - We collect job results so you can work with them later
- Job runtime statistics
  - Runtime duration stats are baked in
- Job timeouts
  - Fine tune timeouts on a per job basis

Example/notes:

```golang
package main

import (
	"fmt"
	"time"
	"github.com/oze4/workerpoolxt"
)

func main() {
	wp := workerpoolxt.New(10, time.Duration(time.Second*10))

	wp.SubmitXT(Job{ // For demo purposes, this job will timeout
		Name:    "My first job",
		Timeout: time.Duration(time.Second * 1),
		//                     ^^^^^^^^^^^^^^^
		Task: func() Response {
			time.Sleep(time.Second * 2)
			//         ^^^^^^^^^^^^^^^
			return Response{Data: "Hello"}
		},
	})

	// Submit as many jobs as you would like

	results := wp.StopWaitXT()

	for _, r := range results {
		fmt.Printf("%s took %fms\n", r.Name(), r.RuntimeDuration() * time.Millisecond)
	}
}
```

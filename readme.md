# reactor

Use like:

```golang
package main

import (
	"fmt"
	"time"

	"github.com/oze4/reactor"
)

func main() {
	// General timeout for jobs
	// You can also supply a timeout per job. If a timeout is not set for a job
	// then we use this general timeout.
	timeoutForJobs := time.Duration(time.Second * 10)
	numOfWorkers := 10

	reactr := reactor.New(numOfWorkers, timeoutForJobs)

	// You can also create a Reactor with a custom Client
	// myreactor := reactor.NewWithClient(numOfWorkers, timeoutForJobs, &reactor.Client{...})

	/**
	 * Add events one by one. 
	 *
	 * Add as many events one by one that you wish, BEFORE calling `myreactor.Reactions()`
	 */

	reactr.React(reactor.NewEvent("event", func(c *reactor.Client) reactor.Reaction {
		// if error, set the error field
		someError := errors.New("some error")
		if someError != nil {
			return reactor.Reaction{Error: someError}
		}
		// if no error, set response field
		yourResult := "pretend I am a data that was returned by something"
		return reactor.Reaction{Response: yourResult}
	}))

	// event with timeout
	reactr.React(reactor.NewEventWithTimeout("eventWithTimeout", time.Duration(time.Second*3), func(c *reactor.Client) reactor.Reaction {
		// ...
	}))

	// etc...

	/**
	 * Adding events in bulk
	 */

	// However you get a slice of events, it doesn't matter	
	manyEvents := []reactor.Event{} 
	// Bulk add
	reactr.Reacts(manyEvents)

	// All results will be here - ReactionsStop gets results and kills workerpool
	// *Events cannot be added after calling ReactionsStop()*
	results := reactr.ReactionsStop()

	// If you do not want to kill wokerpool, and wish to continue to add jobs after
	// getting results, you could also do:
	//
	//// results := reactr.ReactionsWait()
	//
	// Continue adding jobs
	// ...

	for _, result := range results {
		fmt.Println(result)
	}
}
```

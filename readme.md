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

	reactr.React(reactor.NewEvent("name", func(c *reactor.Client) reactor.Reaction {
		return reactor.Reaction{}
	}))

	// etc...

	/**
	 * Add events in bulk
	 *
	 * Keep in mind, instead of using a loop to add events to `manyEvents` you could gather
	 * events in a slice in a number of ways. It doesn't matter how we wind up with a slice
	 * of Events, just that we have a slice of Events.
	 */

	var manyEvents reactor.Events
	for i := 0; i < 10; i++ {
		name := "event " + fmt.Sprintf("%d", i)
		myevent := reactor.NewEvent(name, func(c *reactor.Client) reactor.Reaction {
			return reactor.Reaction{}
		})
		manyEvents = append(manyEvents, myevent)
	}

	// Add multiple
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

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

	myreactor := reactor.New(numOfWorkers, timeoutForJobs)

	// You can also create a Reactor with a custom Client
	// myreactor := reactor.NewWithClient(numOfWorkers, timeoutForJobs, &reactor.Client{...})

	// Add job(s)
	myreactor.Add(reactor.Job{
		Name: "job1",
		Runner: func(c *reactor.Client) reactor.React {
			// do something with client `c`
			res, _ := c.HTTP.Get("xyz.com")
			return reactor.React{Info: res}
		},
	})

	// All results will be here
	results := myreactor.GetResults()
	
	for _, result := range results {
		fmt.Println(result)
	}
}
```

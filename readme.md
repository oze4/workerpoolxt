# workerpoolxt

Worker pool library that extends [github.com/gammazero/workerpool](github.com/gammazero/workerpool).

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

func main() {
	// TODO...
}
```
<p align="center">
  <h1 align="center">workerpoolxt</h1>	
</p>

<p align="center">
  <a href="https://github.com/oze4/workerpoolxt/actions">
    <img title="Build" src="https://github.com/oze4/workerpoolxt/workflows/Build/badge.svg?branch=master" >
  </a>
  <a href="https://codecov.io/gh/oze4/workerpoolxt">
    <img title="codecov" src="https://codecov.io/gh/oze4/workerpoolxt/branch/master/graph/badge.svg" >
  </a>
  <a href="https://goreportcard.com/report/github.com/oze4/workerpoolxt">
    <img title="Go Report Card" src="https://goreportcard.com/badge/github.com/oze4/workerpoolxt" >
  </a>
  <a href="https://github.com/oze4/workerpoolxt/blob/master/LICENSE">
    <img title="License: MIT" src="https://img.shields.io/badge/License-MIT-blue.svg" >
  </a>
  <a href="https://pkg.go.dev/github.com/oze4/workerpoolxt">
    <img title="PkgGoDev" src="https://pkg.go.dev/badge/github.com/oze4/workerpoolxt" >
  </a>
</p>

Worker pool library that extends [github.com/gammazero/workerpool](https://github.com/gammazero/workerpool).

## Synopsis

- Allows you to retain access to underlying `*WorkerPool` object as if you imported `workerpool` directly
- [Full, basic example](#basic-example)
- [How we extend `workerpool`](#how-we-extend-workerpool)
  - [Results](#results)
    - Job results are captured so you can work with them later
    - [How to handle errors?](#error-handling)
  - [Job timeouts](#job-timeouts)
    - Required global/default timeout for all jobs
    - Optional timeout on a per job basis
    - **Job timeout overrides global/default timeout**
    - [How do I know if a job timed out?](#error-handling)
  - [Retry](#retry)
    - `int` that defines N number of retries
    - Can only supply retry on a per job basis
  - [Options](#options)
    - Options are optional
    - Provide either [global/default options](#default-options) or [per job options](#per-job-options)
    - Options are nothing more than `map[string]interface{}` so that you may supply anything you wish
    - Job options override default options, **_we do NOT merge options_**
  - Runtime duration
    - Access a job's runtime duration via it's response
    - e.g. `howLongItTook := someResponseFromSomeJob.RuntimeDuration() //-> time.Duration`

---

## Basic Example

```golang
package main

import (
  "fmt"
  "time"
  wpxt "github.com/oze4/workerpoolxt"
)

func main() {
  defaultTimeout := time.Duration(time.Second*10)
  numWorkers := 10

  wp := wpxt.New(numWorkers, defaultTimeout)

  wp.SubmitXT(wpxt.Job{
    Name: "My first job",
    Task: func(o wpxt.Options) wpxt.Response {
      return wpxt.Response{Data: "Hello, world!"}
    },
  })

  jobResults := wp.StopWaitXT()

  for _, jobresult := range jobResults {
    fmt.Println(jobresult)
  }
}
```

## How we extend `workerpool`

### Results

```golang
// ... jobs submitted here
results := wp.StopWaitXT() // -> []wpxt.Response

for _, result := range results {
  // If job failed, `result.Error != nil`
  // ...
}
```

### Error Handling

- What if I encounter an error in one of my jobs?
- How can I handle or check for it?

```golang
// Just set the `Error` field on the `wpxt.Response` you return
wp.SubmitXT(wpxt.Job{
  Name: "How to handle errors",
  Task: func(o wpxt.Options) wpxt.Response {
    // Pretend we got an error doing something
    if theError != nil {
      return wpxt.Response{Error: theError}
    }
  }
})

results := wp.StopWaitXT()
// Consider `rez` in the following example:
//> for _, rez := range results { ... }
// Check for job error like: `rez.Error != nil`
```

### Job Timeouts

```golang
// Required to set global/default
// timeout when calling `New(...)`
myDefaultTimeout := time.Duration(time.Second*30)
// Or supply per job - if a job has a timeout,
// it overrides the default
wp.SubmitXT(wpxt.Job{
  Name: "Job timeouts",
  // Set timeout field on job
  Timeout: time.Duration(time.Millisecond*500),
  Task: func(o wpxt.Options) wpxt.Response { /* ... */ }
})

results := wp.StopWaitXT()
// ...
```

### Retry

- Seamlessly retry failed jobs
- Optional to provide a Retry int per job

```golang
wp.SubmitXT(wpxt.Job{
  // This job will retry 5 times if failed,
  // as long as we have not exceeded our job timeout
  Retry: 5,
  Name: "I will retry 5 times",
  // Set timeout field on job
  Timeout: time.Duration(time.Millisecond*500),
  Task: func(o wpxt.Options) wpxt.Response {
    return wpxt.Response{Error: errors.New("some_err")}
  },
})

results := wp.StopWaitXT()
// ...
```

### Options

- Help make jobs flexible

#### Default Options

```golang
myopts := map[string]interface{}{
    "myclient": &http.Client{},
}

wp := wpxt.New(10, time.Duration(time.Second*10))
wp.WithOptions(myopts)

wp.SubmitXT(wpxt.Job{
    Name: "myjob",
    Task: func(o wpxt.Options) wpxt.Response {
        // access options here
        client := o["myclient"]
    },
})
```

#### Per Job Options

```golang
myhttpclient := &http.Client{}
myk8sclient := kubernetes.Clientset{}

// This Job Only Needs an HTTP Client
wp.SubmitXT(wpxt.Job{
    Name: "This Job Only Needs an HTTP Client",
    Options: map[string]interface{}{
        "http": myhttpclient,
    },
    Task: func(o wpxt.Options) wpxt.Response {
        // access options here
        httpclient := o["http"]
        // ... do work with `httpclient`
    },
})

// This Job Only Needs Kubernetes Clientset
wp.SubmitXT(wpxt.Job{
    Name: "This Job Only Needs Kubernetes Clientset",
    Options: map[string]interface{}{
        "kube": myk8sclient,
    },
    Task: func(o wpxt.Options) wpxt.Response {
      // access options here
      kubernetesclient := o["kube"]
      // ... do work with `kubernetesclient`
    },
})
```

# WorkQueue library

Library that provides a mechanism for managing and distributing a collection 
of work items among worker threads for processing. The library provides an API for 
adding work items to a queue, starting and stopping worker threads, and querying the state 
of the queue.

check doc in https://pkg.go.dev/github.com/lazaroMB/workqueue

## Usage example
```go
package main

import (
	"time"
	"github.com/lazaroMB/workqueue"
)

type Process struct {
	delay time.Duration
}

func (p *Process) Exec() {
	time.Sleep(p.delay)
}

func main() {
	queue := workqueue.New[*Process](5) // 5 workers

	go queue.StartWorker()

	queue.AddWork(&Process{delay: 10 * time.Second})
}
```

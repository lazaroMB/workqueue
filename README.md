# WorkQueue library

Library that provides a mechanism for managing and distributing a collection 
of work items among worker threads for processing. The library provides an API for 
adding work items to a queue, starting and stopping worker threads, and querying the state 
of the queue.

## Usage example
```
type Process struct {
    delay time.Duration
}

func (p *Process) Exec() {
    time.Sleep(delay time.Second)
}

queue := New[*Process](5) // 5 workers

go queue.StartWorker()

queue.AddWork(&Process{delay: 10})
```

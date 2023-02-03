// Library that provides a mechanism for managing and distributing a collection 
// of work items among worker threads for processing. The library provides an API for 
// adding work items to a queue, starting and stopping worker threads, and querying the state 
// of the queue and its worker threads 
package workqueue

import (
	"errors"
	"math"
)

// WorkItemI interface has a single method named "Exec".
// Any type that implements this interface must have a method named "Exec" 
// with no arguments and no return value.
type WorkItemI interface {
	Exec()
}

// The interface is parameterized by "T WorkItemI", which means that it can work with
// any type "T" that implements the WorkItemI interface
type WorkQueueI[T WorkItemI] interface {
	AddWork(workItem T)
	StartWorker()
	StopWorker()
	GetQueueSize() int
	GetProcessingCount() int
	GetThreadsOccupation() float64
	HasThreadsCapacity() bool
}

type WorkQueue[T WorkItemI] struct {
	queue           []T
	maxThreads      int
	processingCount int
	addQueueChan    chan T
	stopWorkerChan  chan bool
	endJobChan      chan bool
}

// Takes a single argument "workItem T", which represents a work item of type T 
// that implements the WorkItemI interface.
func (w *WorkQueue[T]) AddWork(item T) {
	w.addQueueChan <- item
}

// Starts a worker thread that will take work items from the queue and execute them.
func (w *WorkQueue[T]) StartWorker() {
	runPool := true
	for runPool {
		select {
		case res := <-w.addQueueChan:
			w.addToQueue(res)
			if ok := w.tryToExecNextTask(); ok {
				w.processingCount++
			}
		case <-w.endJobChan:
			w.processingCount--
			if ok := w.tryToExecNextTask(); ok {
				w.processingCount++
			}
		case <-w.stopWorkerChan:
			runPool = false
		}
	}
	defer close(w.endJobChan)
	defer close(w.stopWorkerChan)
	defer close(w.addQueueChan)
}

// Stops the worker thread.
func (w *WorkQueue[T]) StopWorker() {
	w.stopWorkerChan <- true
}

// Returns the size of the work queue as an integer.
func (w *WorkQueue[T]) GetQueueSize() int {
	return len(w.queue)
}

// Returns the number of work items currently being processed by the worker thread as an integer.
func (w *WorkQueue[T]) GetProcessingCount() int {
	return w.processingCount
}

// Returns the occupation rate of the worker thread as a float64 value.
func (w *WorkQueue[T]) GetThreadsOccupation() float64 {
	if w.maxThreads == 0 {
		return 1
	}
	return math.Ceil(float64(w.processingCount)/float64(w.maxThreads)) / 100
}

// Returns a boolean indicating whether the worker thread has capacity to process more work items.
func (w *WorkQueue[T]) HasThreadsCapacity() bool {
	return w.GetThreadsOccupation() < 1
}

func (w *WorkQueue[T]) getNextTask() (T, error) {
	if w.GetQueueSize() == 0 {
		return *new(T), errors.New(EMPTY_QUEUE_ERROR_MSG)
	}
	job := w.queue[0]
	w.queue = w.queue[1:]
	return job, nil
}

func (w *WorkQueue[T]) tryToExecNextTask() bool {
	if !w.HasThreadsCapacity() {
		return false
	}
	nextTask, err := w.getNextTask()
	if err != nil {
		return false
	}
	go w.startJob(nextTask)

	return true
}

func (w *WorkQueue[T]) startJob(job T) {
	job.Exec()
	w.endJobChan <- true
}

func (w *WorkQueue[T]) addToQueue(job T) {
	w.queue = append(w.queue, job)
}

// The function takes a single argument "maxThreads int", which represents the maximum number of worker 
// threads that can be created.
func New[T WorkItemI](maxThreads int) WorkQueueI[T] {
	return &WorkQueue[T]{
		maxThreads:      maxThreads,
		queue:           make([]T, 0),
		processingCount: 0,
		addQueueChan:    make(chan T),
		endJobChan:      make(chan bool),
		stopWorkerChan:  make(chan bool),
	}
}

// Library that provides a mechanism for managing and distributing a collection 
// of work items among worker threads for processing. The library provides an API for 
// adding work items to a queue, starting and stopping worker threads, and querying the state 
// of the queue and its worker threads 
package workqueue

import (
	"errors"
	"sync"
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
	mu              sync.RWMutex
	queue           []T
	maxThreads      int
	processingCount int
	addQueueChan    chan T
	stopWorkerChan  chan bool
	endJobChan      chan bool
	stopped         bool
}

// Takes a single argument "workItem T", which represents a work item of type T 
// that implements the WorkItemI interface.
func (w *WorkQueue[T]) AddWork(item T) {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	w.addQueueChan <- item
}

// Starts a worker thread that will take work items from the queue and execute them.
func (w *WorkQueue[T]) StartWorker() {
	runPool := true
	for runPool {
		select {
		case res := <-w.addQueueChan:
			w.addToQueue(res)
			w.tryToExecNextTask()
		case <-w.endJobChan:
			w.tryToExecNextTask()
		case <-w.stopWorkerChan:
			runPool = false
		}
	}
}

// Stops the worker thread.
func (w *WorkQueue[T]) StopWorker() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.stopped = true
	w.mu.Unlock()

	w.stopWorkerChan <- true
}

// Returns the size of the work queue as an integer.
func (w *WorkQueue[T]) GetQueueSize() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.queue)
}

// Returns the number of work items currently being processed by the worker thread as an integer.
func (w *WorkQueue[T]) GetProcessingCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.processingCount
}

// Returns the occupation rate of the worker thread as a float64 value.
func (w *WorkQueue[T]) GetThreadsOccupation() float64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.maxThreads <= 0 {
		return 1.0
	}
	return float64(w.processingCount) / float64(w.maxThreads)
}

// Returns a boolean indicating whether the worker thread has capacity to process more work items.
func (w *WorkQueue[T]) HasThreadsCapacity() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.maxThreads <= 0 {
		return false
	}
	return w.processingCount < w.maxThreads
}

func (w *WorkQueue[T]) getNextTaskUnlocked() (T, error) {
	if len(w.queue) == 0 {
		return *new(T), errors.New(EMPTY_QUEUE_ERROR_MSG)
	}
	job := w.queue[0]
	w.queue = w.queue[1:]
	return job, nil
}

func (w *WorkQueue[T]) getNextTask() (T, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.getNextTaskUnlocked()
}

func (w *WorkQueue[T]) tryToExecNextTask() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.maxThreads <= 0 || w.processingCount >= w.maxThreads {
		return false
	}
	nextTask, err := w.getNextTaskUnlocked()
	if err != nil {
		return false
	}
	w.processingCount++
	go w.startJob(nextTask)

	return true
}

func (w *WorkQueue[T]) startJob(job T) {
	defer func() {
		w.mu.Lock()
		w.processingCount--
		stopped := w.stopped
		w.mu.Unlock()

		if !stopped {
			select {
			case w.endJobChan <- true:
			default:
				// Non-blocking write to avoid hanging if the worker stopped
			}
		}
	}()
	job.Exec()
}

func (w *WorkQueue[T]) addToQueue(job T) {
	w.mu.Lock()
	defer w.mu.Unlock()
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
		endJobChan:      make(chan bool, maxThreads+1), // Buffered to handle multiple signals without blocking
		stopWorkerChan:  make(chan bool, 1),
	}
}

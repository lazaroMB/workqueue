package workqueue

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockWorkItem struct {
	sleep    time.Duration
	executed atomic.Bool
}

func (m *MockWorkItem) Exec() {
	time.Sleep(m.sleep)
	m.executed.Store(true)
}

func TestAddToQueue(t *testing.T) {
	work := &WorkQueue[*MockWorkItem]{
		maxThreads:      1,
		queue:           make([]*MockWorkItem, 0),
		processingCount: 0,
		addQueueChan:    make(chan *MockWorkItem),
		endJobChan:      make(chan bool),
		stopWorkerChan:  make(chan bool),
	}
	work.addToQueue(&MockWorkItem{sleep: 0})
	assert.Equal(t, work.GetQueueSize(), 1, "This should be 1")
}

func TestGetNextTask(t *testing.T) {
	work := &WorkQueue[*MockWorkItem]{
		maxThreads:      1,
		queue:           make([]*MockWorkItem, 0),
		processingCount: 0,
		addQueueChan:    make(chan *MockWorkItem),
		endJobChan:      make(chan bool),
		stopWorkerChan:  make(chan bool),
	}
	_, err := work.getNextTask()
	assert.Error(t, err)
	assert.Equal(t, err, errors.New(EMPTY_QUEUE_ERROR_MSG))
	work.addToQueue(&MockWorkItem{sleep: 0})
	assert.Equal(t, work.GetQueueSize(), 1, "This should be 1")
	_, err = work.getNextTask()
	assert.Equal(t, work.GetQueueSize(), 0, "This should be 0")
	assert.Equal(t, err, nil)
}

func TestTryExecNextTask(t *testing.T) {
	work := &WorkQueue[*MockWorkItem]{
		maxThreads:      1,
		queue:           make([]*MockWorkItem, 0),
		processingCount: 0,
		addQueueChan:    make(chan *MockWorkItem),
		endJobChan:      make(chan bool),
		stopWorkerChan:  make(chan bool),
	}
	executed := work.tryToExecNextTask()
	assert.Equal(t, executed, false, "This should be false")
	work.addToQueue(&MockWorkItem{sleep: 0})
	executed = work.tryToExecNextTask()
	assert.Equal(t, executed, true, "This should be true")
}

func TestAddWorkToWorkQueue(t *testing.T) {
	work := New[*MockWorkItem](0)
	go work.StartWorker()
	work.AddWork(&MockWorkItem{sleep: 0})
	
	assert.Eventually(t, func() bool {
		return work.GetQueueSize() == 1
	}, 1*time.Second, 10*time.Millisecond, "Queue size should eventually be 1")
}

func TestNoneThreadCapacity(t *testing.T) {
	work := New[*MockWorkItem](0)

	go work.StartWorker()

	assert.Equal(t, work.HasThreadsCapacity(), false, "This must be false")
	assert.Equal(t, work.GetProcessingCount(), 0, "This should be 0")
	assert.Equal(t, work.GetThreadsOccupation(), 1.0, "This should be 1.0")
	
	for i := 0; i < 5; i++ {
		work.AddWork(&MockWorkItem{sleep: 0})
	}
	
	assert.Eventually(t, func() bool {
		return work.GetQueueSize() == 5
	}, 1*time.Second, 10*time.Millisecond, "Queue size should eventually be 5")
	
	assert.Equal(t, work.GetProcessingCount(), 0, "This should be 0")
	assert.Equal(t, work.GetThreadsOccupation(), 1.0, "This should be 1.0")
}

func TestProcessingWithSingleThread(t *testing.T) {
	work := New[*MockWorkItem](1)

	go work.StartWorker()

	assert.Equal(t, work.HasThreadsCapacity(), true, "This must be true")
	assert.Equal(t, work.GetProcessingCount(), 0, "This should be 0")
	assert.Equal(t, work.GetThreadsOccupation(), 0.0, "This should be 0")
	
	jobs := []*MockWorkItem{
		{sleep: 1 * time.Microsecond},
		{sleep: 1 * time.Microsecond},
		{sleep: 1 * time.Microsecond},
		{sleep: 1 * time.Microsecond},
		{sleep: 1 * time.Microsecond},
		{sleep: 1 * time.Microsecond},
	}
	for _, job := range jobs {
		work.AddWork(job)
	}
	
	assert.Eventually(t, func() bool {
		for _, job := range jobs {
			if !job.executed.Load() {
				return false
			}
		}
		return true
	}, 2*time.Second, 10*time.Millisecond, "All jobs should eventually execute")

	assert.Equal(t, work.GetProcessingCount(), 0, "This should be 0")
	assert.Equal(t, work.GetThreadsOccupation(), 0.0, "This should be 0")
}

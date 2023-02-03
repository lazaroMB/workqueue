package workqueue

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockWorkItem struct {
	sleep     time.Duration
	excecuted bool
}

func (m *MockWorkItem) Exec() {
	time.Sleep(m.sleep)
	m.excecuted = true
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
	work.addToQueue(&MockWorkItem{sleep: 2 * time.Second})
	assert.Equal(t, work.GetQueueSize(), 1, "This should be 5")
}

func TestGetNestTask(t *testing.T) {
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
	work.addToQueue(&MockWorkItem{sleep: 2 * time.Second})
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
	excecuted := work.tryToExecNextTask()
	assert.Equal(t, excecuted, false, "This should by false")
	work.addToQueue(&MockWorkItem{sleep: 2 * time.Second})
	excecuted = work.tryToExecNextTask()
	assert.Equal(t, excecuted, true, "This should be true")
}

func TestAddWorkToWorkQueue(t *testing.T) {
	work := New[*MockWorkItem](0)
	go work.StartWorker()
	work.AddWork(&MockWorkItem{sleep: 1 * time.Second})
	time.Sleep(1 * time.Second)
	assert.Equal(t, work.GetQueueSize(), 1, "This should be 5")
}

func TestNoneThreadCapacity(t *testing.T) {
	work := New[*MockWorkItem](0)

	go work.StartWorker()

	assert.Equal(t, work.HasThreadsCapacity(), false, "This must be false")
	assert.Equal(t, work.GetProcessingCount(), 0, "This should be 0")
	assert.Equal(t, work.GetThreadsOccupation(), 1.0, "This should be 1.0")
	for i := 0; i < 5; i++ {
		work.AddWork(&MockWorkItem{sleep: 2 * time.Second})
	}
	time.Sleep(1 * time.Second)
	assert.Equal(t, work.GetProcessingCount(), 0, "This should be 0")
	assert.Equal(t, work.GetThreadsOccupation(), 1.0, "This should be 1.0")
	assert.Equal(t, work.GetQueueSize(), 5, "This should be 5")
}

func TestProcessingWithSingleThread(t *testing.T) {
	work := New[*MockWorkItem](1)

	go work.StartWorker()

	assert.Equal(t, work.HasThreadsCapacity(), true, "This must be true")
	assert.Equal(t, work.GetProcessingCount(), 0, "This should be 0")
	assert.Equal(t, work.GetThreadsOccupation(), 0.0, "This should be 0")
	jobs := []*MockWorkItem{
		{sleep: 2 * time.Nanosecond},
		{sleep: 3 * time.Nanosecond},
		{sleep: 4 * time.Nanosecond},
		{sleep: 5 * time.Nanosecond},
		{sleep: 6 * time.Nanosecond},
		{sleep: 7 * time.Nanosecond},
	}
	for _, job := range jobs {
		work.AddWork(job)
	}
	time.Sleep(1 * time.Second)
	assert.Equal(t, work.GetProcessingCount(), 0, "This should be 0")
	assert.Equal(t, work.GetThreadsOccupation(), 0.0, "This should be 0")
	for _, job := range jobs {
		assert.Equal(t, job.excecuted, true, "This should be true")
	}
}

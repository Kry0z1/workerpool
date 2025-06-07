package workerpool

import (
	"errors"
	"sync"
	"sync/atomic"
)

var ErrStopped = errors.New("WorkerPool is stopped")

type TaskHandler func(workerID int, task string)

// Primitive worker pool. Available to concurrent usage
type WorkerPool struct {
	workerIdGenerator atomic.Int32

	wg sync.WaitGroup

	// chan to signal that worker pool is stopping
	stopping chan struct{}

	done    chan struct{}
	tasks   chan string
	handler TaskHandler
}

func (w *WorkerPool) AddWorker() {
	w.workerIdGenerator.Add(1)
	w.wg.Add(1)

	id := w.workerIdGenerator.Load()

	go func() {
		for {
			select {
			case <-w.stopping:
				w.wg.Done()
				return
			case <-w.done:
				w.wg.Done()
				return
			case task := <-w.tasks:
				w.handler(int(id), task)
			}
		}
	}()
}

// Removes worker from pool.
// If bufferSize == 0 then blocks until at least 1 worker is free to be disposed.
func (w *WorkerPool) RemoveWorker() {
	w.done <- struct{}{}
}

// Submits task directly to the workers.
// Returns ErrStopped on stopped pool.
//
// Is blocking call if bufferSize == 0 or buffer is full.
func (w *WorkerPool) Submit(task string) error {
	select {
	case <-w.stopping:
		return ErrStopped
	case w.tasks <- task:
		return nil
	}
}

// Waits for workers to end their tasks and tasks in buffer and shuts workers down
func (w *WorkerPool) Stop() {
	close(w.stopping)

	w.wg.Wait()
}

// If workers < 1, then no workers are created.
// bufferSize is size of inner task buffer. bufferSize <= 0 <=> no buffering.
func NewWorkerPool(workers int, handler TaskHandler, bufferSize int) *WorkerPool {
	var wp WorkerPool
	if bufferSize <= 0 {
		wp = WorkerPool{
			done:     make(chan struct{}),
			tasks:    make(chan string),
			stopping: make(chan struct{}),
			handler:  handler,
		}
	} else {
		wp = WorkerPool{
			done:     make(chan struct{}, bufferSize),
			tasks:    make(chan string, bufferSize),
			stopping: make(chan struct{}),
			handler:  handler,
		}
	}

	for range workers {
		wp.AddWorker()
	}

	return &wp
}

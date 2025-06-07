package workerpool_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Kry0z1/workerpool"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerPool_BasicFunctionality(t *testing.T) {
	const workers = 3
	const tasks = 5

	var (
		mu      sync.Mutex
		results []string
		wg      sync.WaitGroup
	)

	handler := func(workerID int, task string) {
		mu.Lock()
		defer mu.Unlock()
		results = append(results, task)
		wg.Done()
	}

	pool := workerpool.NewWorkerPool(workers, handler, 1)
	defer pool.Stop()

	wg.Add(tasks)
	for i := range tasks {
		task := string(rune('A' + i))
		err := pool.Submit(task)
		require.NoError(t, err)
	}

	wg.Wait()
	assert.Len(t, results, tasks)
}

func TestWorkerPool_Stop(t *testing.T) {
	handler := func(workerID int, task string) {
		time.Sleep(50 * time.Millisecond)
	}

	pool := workerpool.NewWorkerPool(2, handler, 1)

	for i := range 3 {
		err := pool.Submit(string(rune('A' + i)))
		require.NoError(t, err)
	}

	pool.Stop()
	err := pool.Submit("should fail")
	assert.ErrorIs(t, err, workerpool.ErrStopped)
}

func TestWorkerPool_RemoveWorker(t *testing.T) {
	const initialWorkers = 3
	var activeWorkers atomic.Int32

	blocker := make(chan struct{})
	handler := func(workerID int, task string) {
		activeWorkers.Add(1)
		defer activeWorkers.Add(-1)
		<-blocker
	}

	pool := workerpool.NewWorkerPool(initialWorkers, handler, 1)

	for range initialWorkers {
		err := pool.Submit("block")
		require.NoError(t, err)
	}

	time.Sleep(100 * time.Millisecond)

	blocker <- struct{}{}
	pool.RemoveWorker()

	assert.Eventually(t, func() bool {
		return int(activeWorkers.Load()) == initialWorkers-1
	}, 1*time.Second, 50*time.Millisecond)
}

func TestWorkerPool_ConcurrentProcessing(t *testing.T) {
	const workers = 4
	const tasks = 20

	var (
		workerStats     = make(map[int]int)
		mu              sync.Mutex
		concurrentCount atomic.Int32
		maxConcurrent   atomic.Int32
	)

	handler := func(workerID int, task string) {
		current := concurrentCount.Add(1)
		defer concurrentCount.Add(-1)

		for {
			old := maxConcurrent.Load()
			if current > old && !maxConcurrent.CompareAndSwap(old, current) {
				continue
			}
			break
		}

		mu.Lock()
		workerStats[workerID]++
		mu.Unlock()

		time.Sleep(10 * time.Millisecond)
	}

	pool := workerpool.NewWorkerPool(workers, handler, 1)
	defer pool.Stop()

	var wg sync.WaitGroup
	for i := range tasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool.Submit(string(rune('A' + i)))
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(workers), maxConcurrent.Load())

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, workerStats, workers)
	for _, count := range workerStats {
		assert.Greater(t, count, 0, "All workers should process tasks")
	}
}

func TestWorkerPool_SubmitAfterStop(t *testing.T) {
	pool := workerpool.NewWorkerPool(1, func(int, string) {}, 0)
	pool.Stop()

	err := pool.Submit("test")
	assert.ErrorIs(t, err, workerpool.ErrStopped)
}

func TestWorkerPool_ZeroWorkers(t *testing.T) {
	processed := make(chan struct{})
	handler := func(int, string) {
		close(processed)
	}

	pool := workerpool.NewWorkerPool(0, handler, 1)

	go func() {
		pool.Submit("test")
	}()

	select {
	case <-processed:
		t.Fatal("Task processed with zero workers")
	case <-time.After(100 * time.Millisecond):
	}
}

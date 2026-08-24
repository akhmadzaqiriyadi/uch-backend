package worker

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPool_SubmitAndProcess(t *testing.T) {
	pool := NewPool(3, 10)
	defer pool.Shutdown()

	var counter int64
	var wg sync.WaitGroup
	numTasks := 10

	wg.Add(numTasks)
	for i := 0; i < numTasks; i++ {
		submitted := pool.Submit(func(ctx context.Context) error {
			atomic.AddInt64(&counter, 1)
			wg.Done()
			return nil
		})
		assert.True(t, submitted)
	}

	wg.Wait()
	assert.Equal(t, int64(numTasks), atomic.LoadInt64(&counter))
}

func TestPool_QueueFullDrop(t *testing.T) {
	// 1 worker, buffer size 1
	pool := NewPool(1, 1)
	defer pool.Shutdown()

	started := make(chan struct{})
	blockerDone := make(chan struct{})

	// Submit blocker task
	pool.Submit(func(ctx context.Context) error {
		close(started)
		<-blockerDone
		return nil
	})

	// Wait until worker is actively executing the blocker
	<-started

	// Fill the 1-slot buffer
	ok1 := pool.Submit(func(ctx context.Context) error { return nil })
	assert.True(t, ok1)

	// Next submit should drop immediately because buffer is full
	ok2 := pool.Submit(func(ctx context.Context) error { return nil })
	assert.False(t, ok2)

	close(blockerDone)
}

func TestPool_GracefulShutdown(t *testing.T) {
	pool := NewPool(2, 5)
	var processed int64

	pool.Submit(func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt64(&processed, 1)
		return nil
	})

	pool.Shutdown()
	assert.Equal(t, int64(1), atomic.LoadInt64(&processed))
}

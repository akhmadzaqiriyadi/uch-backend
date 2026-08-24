package worker

import (
	"context"
	"log/slog"
	"sync"
)

// Task represents an asynchronous unit of work
type Task func(ctx context.Context) error

// Pool represents a worker pool for running background jobs
type Pool struct {
	tasks      chan Task
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	numWorkers int
}

// NewPool initializes a new worker pool
// numWorkers: Number of concurrent worker goroutines
// bufferSize: Maximum queued tasks in channel
func NewPool(numWorkers, bufferSize int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		tasks:      make(chan Task, bufferSize),
		numWorkers: numWorkers,
		ctx:        ctx,
		cancel:     cancel,
	}

	p.start()
	return p
}

func (p *Pool) start() {
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go func(workerID int) {
			defer p.wg.Done()
			for task := range p.tasks {
				if err := task(p.ctx); err != nil {
					slog.Error("Background task execution failed",
						slog.Int("worker_id", workerID),
						slog.String("error", err.Error()),
					)
				}
			}
		}(i + 1)
	}
}

// Submit queues a task for background processing (non-blocking)
func (p *Pool) Submit(task Task) bool {
	select {
	case p.tasks <- task:
		return true
	default:
		slog.Warn("Worker pool task queue full, task dropped")
		return false
	}
}

// Shutdown gracefully waits for ongoing tasks and stops workers
func (p *Pool) Shutdown() {
	close(p.tasks)
	p.wg.Wait()
	p.cancel()
}

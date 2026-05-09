// Package worker provides the worker pool for async order processing.
// Orders are submitted to the pool and processed concurrently by a fixed
// number of goroutines. This keeps HTTP handlers fast — they return 202
// immediately while workers process orders in the background.
package worker

import (
	"context"
	"sync"

	"retail-platform/order/internal/domain"
	"retail-platform/pkg/logger"
)

// Job represents a single order processing task.
type Job struct {
	OrderID string
	UserID  string
}

// WorkerPool manages a fixed pool of goroutines that process orders.
type WorkerPool struct {
	jobs      chan Job        // Buffered channel of jobs
	wg        sync.WaitGroup  // Tracks active workers
	processor *OrderProcessor // Processes each order
	size      int             // Number of worker goroutines to launch
	log       *logger.Logger
}

// NewWorkerPool creates a new WorkerPool.
// size is the number of worker goroutines — from WORKER_POOL_SIZE config.
// Buffer is size*2 so HTTP handlers can submit jobs without blocking
// even when all workers are busy.
func NewWorkerPool(size int, processor *OrderProcessor, log *logger.Logger) *WorkerPool {
	return &WorkerPool{
		jobs:      make(chan Job, size*2),
		processor: processor,
		size:      size,
		log:       log,
	}
}

// Start launches the worker goroutines.
// Must be called BEFORE the HTTP server starts accepting requests.
// Workers run until the jobs channel is closed by Shutdown.
func (p *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < p.size; i++ {
		p.wg.Add(1)
		go func(workerID int) {
			defer p.wg.Done()
			p.log.Info().Int("worker_id", workerID).Msg("worker started")

			for job := range p.jobs {
				p.log.Info().
					Int("worker_id", workerID).
					Str("order_id", job.OrderID).
					Msg("processing order")

				if err := p.processor.ProcessOrder(ctx, job.OrderID); err != nil {
					p.log.Error().
						Err(err).
						Int("worker_id", workerID).
						Str("order_id", job.OrderID).
						Msg("order processing failed")
				}
			}

			p.log.Info().Int("worker_id", workerID).Msg("worker stopped")
		}(i)
	}
}

// Submit adds a job to the queue.
// Non-blocking — returns ErrPoolFull immediately if the queue is full.
// HTTP handlers must never block waiting for a worker to free up.
func (p *WorkerPool) Submit(job Job) error {
	select {
	case p.jobs <- job:
		return nil
	default:
		return domain.ErrPoolFull
	}
}

// Shutdown drains the job channel and waits for all workers to finish.
// Called AFTER the HTTP server stops accepting requests.
// Guarantees no orders are lost mid-processing during deployments.
func (p *WorkerPool) Shutdown() {
	p.log.Info().Msg("worker pool shutting down — draining jobs")
	close(p.jobs)
	p.wg.Wait()
	p.log.Info().Msg("worker pool stopped cleanly")
}

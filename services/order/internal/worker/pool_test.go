// Package worker — tests for WorkerPool.
// Tests run with -race to detect concurrent access bugs.
package worker

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"retail-platform/order/internal/domain"
	"retail-platform/pkg/logger"
)

// mockProcessor is a test double for OrderProcessor.
// Counts how many orders were processed and simulates work.
type mockProcessor struct {
	processed atomic.Int64
	delay     time.Duration
}

func (m *mockProcessor) ProcessOrder(ctx context.Context, orderID string) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.processed.Add(1)
	return nil
}

func newTestPool(size int, processor *OrderProcessor) *WorkerPool {
	log := logger.New("worker-test")
	return NewWorkerPool(size, processor, log)
}

func TestWorkerPool_Submit(t *testing.T) {
	t.Run("submitted jobs are processed", func(t *testing.T) {
		log := logger.New("worker-test")
		var count atomic.Int64

		processor := &OrderProcessor{
			log: log,
		}
		_ = processor

		pool := &WorkerPool{
			jobs: make(chan Job, 20),
			size: 2,
			log:  log,
			processor: &OrderProcessor{
				log: log,
			},
		}

		// Override processor with a counting one
		var wg sync.WaitGroup
		jobCount := 5
		wg.Add(jobCount)

		pool.processor = &OrderProcessor{log: log}

		ctx := context.Background()

		// Start workers manually with counting logic
		for i := 0; i < pool.size; i++ {
			pool.wg.Add(1)
			go func() {
				defer pool.wg.Done()
				for range pool.jobs {
					count.Add(1)
					wg.Done()
				}
			}()
		}

		// Submit jobs
		for i := 0; i < jobCount; i++ {
			err := pool.Submit(Job{OrderID: "order-" + string(rune('0'+i))})
			if err != nil {
				t.Errorf("unexpected error submitting job: %v", err)
			}
		}

		// Wait for all jobs to be processed
		wg.Wait()

		if count.Load() != int64(jobCount) {
			t.Errorf("expected %d jobs processed, got %d", jobCount, count.Load())
		}

		close(pool.jobs)
		pool.wg.Wait()
	})
}

func TestWorkerPool_ErrPoolFull(t *testing.T) {
	t.Run("returns ErrPoolFull when queue is full", func(t *testing.T) {
		log := logger.New("worker-test")
		pool := &WorkerPool{
			jobs: make(chan Job, 1), // buffer of 1
			size: 1,
			log:  log,
		}

		// Fill the buffer
		err := pool.Submit(Job{OrderID: "order-1"})
		if err != nil {
			t.Fatalf("first submit should succeed: %v", err)
		}

		// Next submit should fail
		err = pool.Submit(Job{OrderID: "order-2"})
		if err != domain.ErrPoolFull {
			t.Errorf("expected ErrPoolFull, got %v", err)
		}
	})
}

func TestWorkerPool_GracefulShutdown(t *testing.T) {
	t.Run("all jobs complete before shutdown returns", func(t *testing.T) {
		log := logger.New("worker-test")
		var processed atomic.Int64

		pool := &WorkerPool{
			jobs: make(chan Job, 10),
			size: 2,
			log:  log,
		}

		ctx := context.Background()

		// Start workers that count processed jobs
		for i := 0; i < pool.size; i++ {
			pool.wg.Add(1)
			go func() {
				defer pool.wg.Done()
				for range pool.jobs {
					time.Sleep(10 * time.Millisecond)
					processed.Add(1)
				}
			}()
		}

		// Submit jobs
		jobCount := 5
		for i := 0; i < jobCount; i++ {
			_ = pool.Submit(Job{OrderID: "order-" + string(rune('0'+i))})
		}

		// Shutdown — should wait for all jobs
		pool.Shutdown()
		_ = ctx

		if processed.Load() != int64(jobCount) {
			t.Errorf("expected %d jobs processed before shutdown, got %d", jobCount, processed.Load())
		}
	})
}

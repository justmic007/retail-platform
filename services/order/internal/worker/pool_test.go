// Package worker — tests for WorkerPool.
// Tests run with -race to detect concurrent access bugs.
package worker

import (
	"context"
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

func newTestPool(size int, processor *mockProcessor) *WorkerPool {
	log := logger.New("worker-test")
	pool := &WorkerPool{
		jobs: make(chan Job, size*10),
		size: size,
		log:  log,
	}
	// wire the mock processor into the pool's worker loop
	for i := 0; i < size; i++ {
		pool.wg.Add(1)
		go func() {
			defer pool.wg.Done()
			for job := range pool.jobs {
				_ = processor.ProcessOrder(context.Background(), job.OrderID)
			}
		}()
	}
	return pool
}

func TestWorkerPool_Submit(t *testing.T) {
	t.Run("submitted jobs are processed", func(t *testing.T) {
		processor := &mockProcessor{}
		pool := newTestPool(2, processor)

		jobCount := 5
		for i := 0; i < jobCount; i++ {
			err := pool.Submit(Job{OrderID: "order-" + string(rune('0'+i))})
			if err != nil {
				t.Errorf("unexpected error submitting job: %v", err)
			}
		}

		pool.Shutdown()

		if processor.processed.Load() != int64(jobCount) {
			t.Errorf("expected %d jobs processed, got %d", jobCount, processor.processed.Load())
		}
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
		processor := &mockProcessor{delay: 10 * time.Millisecond}
		pool := newTestPool(2, processor)

		jobCount := 5
		for i := 0; i < jobCount; i++ {
			_ = pool.Submit(Job{OrderID: "order-" + string(rune('0'+i))})
		}

		// Shutdown — should wait for all in-flight jobs to complete
		pool.Shutdown()

		if processor.processed.Load() != int64(jobCount) {
			t.Errorf("expected %d jobs processed before shutdown, got %d", jobCount, processor.processed.Load())
		}
	})
}

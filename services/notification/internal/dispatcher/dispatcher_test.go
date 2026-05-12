// Package dispatcher — tests for the Dispatcher.
package dispatcher

import (
	"context"
	"testing"
	"time"

	"retail-platform/notification/internal/handler"
	"retail-platform/pkg/events"
	"retail-platform/pkg/logger"
)

// newTestDispatcher creates a dispatcher wired with real handlers for testing.
func newTestDispatcher(bus *events.Bus) *Dispatcher {
	log := logger.New("notification-test")
	email := handler.NewEmailHandler("", "", "", log)
	internal := handler.NewInternalHandler("", "", "", "warehouse@test.com", log)
	return NewDispatcher(bus, email, internal, log)
}

func TestDispatcher_OrderConfirmed(t *testing.T) {
	bus := events.NewBus()
	d := newTestDispatcher(bus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.Run(ctx)

	bus.Orders <- events.OrderEvent{
		Type:      events.EventOrderConfirmed,
		OrderID:   "order-123",
		UserID:    "user-456",
		UserEmail: "micah@example.com",
		Total:     254.95,
	}

	// Give dispatcher time to process
	time.Sleep(50 * time.Millisecond)
}

func TestDispatcher_StockLow(t *testing.T) {
	bus := events.NewBus()
	d := newTestDispatcher(bus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.Run(ctx)

	bus.Stock <- events.StockEvent{
		Type:        events.EventStockLow,
		ProductID:   "product-123",
		ProductName: "Sunflower Oil 2L",
		StockLevel:  3,
	}

	time.Sleep(50 * time.Millisecond)
}

func TestDispatcher_PanicRecovery(t *testing.T) {
	bus := events.NewBus()
	log := logger.New("notification-test")

	// EmailHandler that panics
	panicEmail := &panicEmailHandler{}
	internal := handler.NewInternalHandler("", "", "", "warehouse@test.com", log)
	d := NewDispatcher(bus, panicEmail, internal, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.Run(ctx)

	// Send event that will cause a panic in the handler
	bus.Orders <- events.OrderEvent{
		Type:    events.EventOrderConfirmed,
		OrderID: "order-panic",
	}

	// Give dispatcher time to recover
	time.Sleep(50 * time.Millisecond)

	// Dispatcher must still be alive — send another event
	bus.Orders <- events.OrderEvent{
		Type:    events.EventOrderFailed,
		OrderID: "order-after-panic",
	}

	time.Sleep(50 * time.Millisecond)
	// If we reach here, dispatcher survived the panic ✓
}

func TestDispatcher_GracefulShutdown(t *testing.T) {
	bus := events.NewBus()
	d := newTestDispatcher(bus)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		d.Run(ctx)
		close(done)
	}()

	// Cancel context — dispatcher should stop
	cancel()

	select {
	case <-done:
		// Dispatcher stopped cleanly ✓
	case <-time.After(1 * time.Second):
		t.Error("dispatcher did not stop within 1 second after context cancellation")
	}
}

// panicEmailHandler is a test double that panics on every call.
type panicEmailHandler struct{}

func (p *panicEmailHandler) SendOrderConfirmation(ctx context.Context, event events.OrderEvent) {
	panic("simulated handler panic")
}

func (p *panicEmailHandler) SendOrderFailed(ctx context.Context, event events.OrderEvent) {
	panic("simulated handler panic")
}

func (p *panicEmailHandler) SendOrderCancelled(ctx context.Context, event events.OrderEvent) {
	panic("simulated handler panic")
}

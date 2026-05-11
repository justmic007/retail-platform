// Package dispatcher contains the event dispatcher for the Notification Service.
// It reads from both the Orders and Stock channels on the shared event bus
// using a fan-in select pattern and routes events to the appropriate handlers.
package dispatcher

import (
	"context"

	"retail-platform/notification/internal/handler"
	"retail-platform/pkg/events"
	"retail-platform/pkg/logger"
)

// Dispatcher reads events from the event bus and routes them to handlers.
type Dispatcher struct {
	bus      *events.Bus
	email    emailSender
	internal *handler.InternalHandler
	log      *logger.Logger
}

// emailSender is satisfied by EmailHandler and any test double.
type emailSender interface {
	SendOrderConfirmation(event events.OrderEvent)
	SendOrderFailed(event events.OrderEvent)
	SendOrderCancelled(event events.OrderEvent)
}

// NewDispatcher creates a new Dispatcher.
func NewDispatcher(
	bus *events.Bus,
	email emailSender,
	internal *handler.InternalHandler,
	log *logger.Logger,
) *Dispatcher {
	return &Dispatcher{
		bus:      bus,
		email:    email,
		internal: internal,
		log:      log,
	}
}

// Run blocks until ctx is cancelled. Call as: go dispatcher.Run(ctx)
// Fan-in select reads Orders and Stock channels simultaneously.
// Finishes the current event before exiting — no events dropped.
func (d *Dispatcher) Run(ctx context.Context) {
	d.log.Info().Msg("dispatcher started")

	for {
		select {
		case event, ok := <-d.bus.Orders:
			if !ok {
				d.log.Info().Msg("orders channel closed — dispatcher stopping")
				return
			}
			d.safeHandle(func() { d.handleOrderEvent(event) })

		case event, ok := <-d.bus.Stock:
			if !ok {
				d.log.Info().Msg("stock channel closed — dispatcher stopping")
				return
			}
			d.safeHandle(func() { d.handleStockEvent(event) })

		case <-ctx.Done():
			d.log.Info().Msg("context cancelled — dispatcher stopping")
			return
		}
	}
}

// handleOrderEvent routes an order event to the correct email handler method.
func (d *Dispatcher) handleOrderEvent(event events.OrderEvent) {
	d.log.Info().
		Str("type", string(event.Type)).
		Str("order_id", event.OrderID).
		Msg("handling order event")

	switch event.Type {
	case events.EventOrderConfirmed:
		d.email.SendOrderConfirmation(event)
	case events.EventOrderFailed:
		d.email.SendOrderFailed(event)
	case events.EventOrderCancelled:
		d.email.SendOrderCancelled(event)
	default:
		d.log.Warn().
			Str("type", string(event.Type)).
			Msg("unknown order event type — skipping")
	}
}

// handleStockEvent routes a stock event to the correct internal handler method.
func (d *Dispatcher) handleStockEvent(event events.StockEvent) {
	d.log.Info().
		Str("type", string(event.Type)).
		Str("product_id", event.ProductID).
		Msg("handling stock event")

	switch event.Type {
	case events.EventStockLow:
		d.internal.SendLowStockAlert(event)
	default:
		d.log.Warn().
			Str("type", string(event.Type)).
			Msg("unknown stock event type — skipping")
	}
}

// safeHandle wraps a handler call with panic recovery.
// A panicking handler must not crash the entire dispatcher goroutine.
// The panic is logged and the dispatcher continues processing the next event.
func (d *Dispatcher) safeHandle(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			d.log.Error().
				Interface("panic", r).
				Msg("dispatcher recovered from handler panic")
		}
	}()
	fn()
}

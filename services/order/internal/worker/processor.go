// Package worker — order processor.
// ProcessOrder is called by each worker goroutine for every order job.
// It fetches prices from Inventory Service, reserves stock, and updates
// the order status. On failure it releases any reserved stock.
package worker

import (
	"context"
	"fmt"
	"time"

	"retail-platform/order/internal/client"
	"retail-platform/order/internal/domain"
	"retail-platform/order/internal/repository"
	"retail-platform/pkg/events"
	"retail-platform/pkg/logger"

	"github.com/shopspring/decimal"
)

// reservedItem tracks a successfully reserved stock item.
// Used to release stock if order processing fails partway through.
type reservedItem struct {
	productID string
	quantity  int
}

// OrderProcessor handles the business logic of processing a single order.
type OrderProcessor struct {
	repo            repository.OrderRepository
	inventoryClient client.InventoryClientInterface
	eventBus        events.Publisher
	log             *logger.Logger
}

// NewOrderProcessor creates a new OrderProcessor.
func NewOrderProcessor(
	repo repository.OrderRepository,
	inventoryClient client.InventoryClientInterface,
	eventBus events.Publisher,
	log *logger.Logger,
) *OrderProcessor {
	return &OrderProcessor{
		repo:            repo,
		inventoryClient: inventoryClient,
		eventBus:        eventBus,
		log:             log,
	}
}

// ProcessOrder processes a single order:
//  1. Update status → PROCESSING
//  2. Fetch price + name for each item from Inventory Service
//  3. Calculate total_amount
//  4. Reserve stock for each item
//  5. Update status → CONFIRMED with total_amount
//  6. Publish OrderConfirmed event
//
// On any failure:
//   - Release any stock that was already reserved
//   - Update status → FAILED
//   - Publish OrderFailed event
func (p *OrderProcessor) ProcessOrder(ctx context.Context, orderID string) error {
	p.log.Info().Str("order_id", orderID).Msg("starting order processing")

	// Step 1 — Mark as PROCESSING so we know it's been picked up
	if err := p.repo.UpdateStatus(ctx, orderID, domain.StatusProcessing); err != nil {
		return fmt.Errorf("update status to processing: %w", err)
	}

	// Step 2 — Fetch the order to get items
	// We use a system context here — no user ownership check needed,
	// the worker is an internal process not a user request
	order, err := p.fetchOrder(ctx, orderID)
	if err != nil {
		p.failOrder(ctx, orderID, "", err)
		return err
	}

	// Step 3 — Fetch price + name for each item from Inventory Service
	// and calculate total_amount
	var totalAmount decimal.Decimal
	var reserved []reservedItem

	for _, item := range order.Items {
		product, err := p.inventoryClient.GetProduct(ctx, item.ProductID)
		if err != nil {
			p.releaseAll(ctx, reserved, orderID)
			p.failOrder(ctx, orderID, order.UserID, err)
			return fmt.Errorf("get product %s: %w", item.ProductID, err)
		}

		// Snapshot price and name at order time
		item.ProductName = product.Name
		item.UnitPrice = product.Price
		item.TotalPrice = product.Price.Mul(decimal.NewFromInt(int64(item.Quantity)))
		totalAmount = totalAmount.Add(item.TotalPrice)
	}

	// Step 4 — Persist price snapshots to order_items
	if err := p.repo.UpdateItems(ctx, order.Items); err != nil {
		p.releaseAll(ctx, reserved, orderID)
		p.failOrder(ctx, orderID, order.UserID, err)
		return fmt.Errorf("update order items: %w", err)
	}

	// Step 5 — Reserve stock for each item
	for _, item := range order.Items {
		if err := p.inventoryClient.Reserve(ctx, item.ProductID, item.Quantity, orderID); err != nil {
			// Release all previously reserved items
			p.releaseAll(ctx, reserved, orderID)
			p.failOrder(ctx, orderID, order.UserID, err)
			return fmt.Errorf("reserve stock for product %s: %w", item.ProductID, err)
		}
		reserved = append(reserved, reservedItem{
			productID: item.ProductID,
			quantity:  item.Quantity,
		})
	}

	// Step 6 — Update status → CONFIRMED with calculated total
	if err := p.repo.UpdateStatusAndTotal(ctx, orderID, domain.StatusConfirmed, totalAmount); err != nil {
		p.releaseAll(ctx, reserved, orderID)
		p.failOrder(ctx, orderID, order.UserID, err)
		return fmt.Errorf("update status to confirmed: %w", err)
	}

	// Step 7 — Publish OrderConfirmed event
	p.publishOrderEvent(ctx, events.EventOrderConfirmed, orderID, order.UserID, totalAmount)

	p.log.Info().
		Str("order_id", orderID).
		Float64("total_amount", totalAmount.InexactFloat64()).
		Msg("order confirmed successfully")

	return nil
}

// fetchOrder retrieves an order by ID without ownership check.
// Used internally by the worker — not a user-facing operation.
func (p *OrderProcessor) fetchOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	order, err := p.repo.FindByIDInternal(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("fetch order: %w", err)
	}
	return order, nil
}

// releaseAll releases stock for all previously reserved items.
// Called when processing fails partway through reservations.
func (p *OrderProcessor) releaseAll(ctx context.Context, reserved []reservedItem, orderID string) {
	for _, item := range reserved {
		if err := p.inventoryClient.Release(ctx, item.productID, item.quantity, orderID); err != nil {
			p.log.Error().
				Err(err).
				Str("order_id", orderID).
				Str("product_id", item.productID).
				Msg("failed to release stock — manual intervention may be needed")
		}
	}
}

// failOrder updates the order status to FAILED and publishes an event.
func (p *OrderProcessor) failOrder(ctx context.Context, orderID, userID string, reason error) {
	p.log.Error().Err(reason).Str("order_id", orderID).Msg("order failed")

	if err := p.repo.UpdateStatus(ctx, orderID, domain.StatusFailed); err != nil {
		p.log.Error().Err(err).Str("order_id", orderID).Msg("failed to update order status to FAILED")
	}

	p.publishOrderEvent(ctx, events.EventOrderFailed, orderID, userID, decimal.Zero)
}

// publishOrderEvent publishes an order event to the event bus non-blocking.
// If the publish fails the error is logged — notifications are best-effort.
func (p *OrderProcessor) publishOrderEvent(ctx context.Context, eventType events.EventType, orderID, userID string, total decimal.Decimal) {
	if err := p.eventBus.PublishOrder(ctx, events.OrderEvent{
		Type:       eventType,
		OrderID:    orderID,
		UserID:     userID,
		Total:      total.InexactFloat64(),
		OccurredAt: time.Now(),
	}); err != nil {
		p.log.Warn().Err(err).Str("order_id", orderID).Msg("failed to publish order event")
	}
}

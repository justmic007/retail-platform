// Package repository — Postgres implementation of OrderRepository.
// This is the ONLY file in the Order Service that contains SQL.
package repository

import (
	"context"
	"errors"
	"fmt"

	"retail-platform/order/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type postgresOrderRepo struct {
	db *pgxpool.Pool
}

// NewPostgresOrderRepo creates a new Postgres order repository.
func NewPostgresOrderRepo(db *pgxpool.Pool) OrderRepository {
	return &postgresOrderRepo{db: db}
}

// Create inserts a new order and its items in a single transaction.
// Either both the order and all items are inserted, or nothing is —
// no partial orders in the database.
func (r *postgresOrderRepo) Create(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Insert the order — RETURNING gives us DB-generated id and timestamps
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (user_id, status, total_amount, idempotency_key, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, status, total_amount, idempotency_key, notes, created_at, updated_at
	`,
		order.UserID,
		order.Status,
		order.TotalAmount,
		order.IdempotencyKey,
		order.Notes,
	).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.TotalAmount,
		&order.IdempotencyKey,
		&order.Notes,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert order: %w", err)
	}

	// Insert all order items
	for _, item := range order.Items {
		err = tx.QueryRow(ctx, `
			INSERT INTO order_items (order_id, product_id, product_name, quantity, unit_price, total_price)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, created_at
		`,
			order.ID,
			item.ProductID,
			item.ProductName,
			item.Quantity,
			item.UnitPrice,
			item.TotalPrice,
		).Scan(&item.ID, &item.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("insert order item: %w", err)
		}
		item.OrderID = order.ID
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit order: %w", err)
	}

	return order, nil
}

// FindByIDInternal retrieves an order by ID only — no ownership check.
// Used by the worker processor — internal process, not a user request.
func (r *postgresOrderRepo) FindByIDInternal(ctx context.Context, orderID string) (*domain.Order, error) {
	order := &domain.Order{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, status, payment_status, total_amount, idempotency_key, notes, created_at, updated_at
		FROM orders
		WHERE id = $1
	`, orderID).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.PaymentStatus,
		&order.TotalAmount,
		&order.IdempotencyKey,
		&order.Notes,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("find order by id internal: %w", err)
	}

	items, err := r.findItemsByOrderID(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	order.Items = items

	return order, nil
}

// FindByID retrieves an order by ID and userID.
// Ownership enforced in SQL — user A cannot see user B's orders.
func (r *postgresOrderRepo) FindByID(ctx context.Context, orderID, userID string) (*domain.Order, error) {
	order := &domain.Order{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, status, payment_status, total_amount, idempotency_key, notes, created_at, updated_at
		FROM orders
		WHERE id = $1 AND user_id = $2
	`, orderID, userID).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.PaymentStatus,
		&order.TotalAmount,
		&order.IdempotencyKey,
		&order.Notes,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("find order by id: %w", err)
	}

	// Fetch order items
	items, err := r.findItemsByOrderID(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	order.Items = items

	return order, nil
}

// FindByUserID returns all orders for a user, newest first.
func (r *postgresOrderRepo) FindByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, status, payment_status, total_amount, idempotency_key, notes, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("find orders by user id: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		order := &domain.Order{}
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Status,
			&order.PaymentStatus,
			&order.TotalAmount,
			&order.IdempotencyKey,
			&order.Notes,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// UpdateItems persists price and name snapshots for all order items.
func (r *postgresOrderRepo) UpdateItems(ctx context.Context, items []*domain.OrderItem) error {
	for _, item := range items {
		_, err := r.db.Exec(ctx, `
			UPDATE order_items
			SET product_name = $1, unit_price = $2, total_price = $3
			WHERE id = $4
		`, item.ProductName, item.UnitPrice, item.TotalPrice, item.ID)
		if err != nil {
			return fmt.Errorf("update order item %s: %w", item.ID, err)
		}
	}
	return nil
}

// UpdateStatus updates the status of an order.
func (r *postgresOrderRepo) UpdateStatus(ctx context.Context, orderID string, status domain.OrderStatus) error {
	_, err := r.db.Exec(ctx, `
		UPDATE orders SET status = $1 WHERE id = $2
	`, status, orderID)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	return nil
}

// UpdateStatusAndTotal updates status and total_amount together.
// Called when order is CONFIRMED with the final calculated total.
func (r *postgresOrderRepo) UpdateStatusAndTotal(ctx context.Context, orderID string, status domain.OrderStatus, total decimal.Decimal) error {
	_, err := r.db.Exec(ctx, `
		UPDATE orders SET status = $1, total_amount = $2 WHERE id = $3
	`, status, total, orderID)
	if err != nil {
		return fmt.Errorf("update order status and total: %w", err)
	}
	return nil
}

// FindByIdempotencyKey looks up an order by idempotency key and userID.
// Returns nil, nil if not found — not an error, just means it's a new order.
func (r *postgresOrderRepo) FindByIdempotencyKey(ctx context.Context, key, userID string) (*domain.Order, error) {
	order := &domain.Order{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, status, payment_status, total_amount, idempotency_key, notes, created_at, updated_at
		FROM orders
		WHERE idempotency_key = $1 AND user_id = $2
	`, key, userID).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.PaymentStatus,
		&order.TotalAmount,
		&order.IdempotencyKey,
		&order.Notes,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find order by idempotency key: %w", err)
	}

	return order, nil
}

// findItemsByOrderID is a private helper that fetches all items for an order.
func (r *postgresOrderRepo) findItemsByOrderID(ctx context.Context, orderID string) ([]*domain.OrderItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, order_id, product_id, product_name, quantity, unit_price, total_price, created_at
		FROM order_items
		WHERE order_id = $1
	`, orderID)
	if err != nil {
		return nil, fmt.Errorf("find order items: %w", err)
	}
	defer rows.Close()

	var items []*domain.OrderItem
	for rows.Next() {
		item := &domain.OrderItem{}
		err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPrice,
			&item.TotalPrice,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

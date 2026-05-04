// Package repository — Postgres implementations of inventory repositories.
// This is the ONLY file in the Inventory Service that contains SQL.
package repository

import (
	"context"
	"errors"
	"fmt"

	"retail-platform/inventory/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Product Repository ────────────────────────────────────────────────────────
type postgresProductRepo struct {
	db *pgxpool.Pool
}

// NewPostgresProductRepo creates a new Postgres product repository
func NewPostgresProductRepo(db *pgxpool.Pool) ProductRepository {
	return &postgresProductRepo{db: db}
}

func (r *postgresProductRepo) List(ctx context.Context) ([]*domain.Product, error) {
	query := `
	 SELECT id, sku, name, description, price, category, is_active, created_at, updated_at
	 FROM products
	 WHERE is_active = true
	 ORDER BY category, name
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		p := &domain.Product{}
		err := rows.Scan(
			&p.ID, &p.SKU, &p.Name, &p.Description,
			&p.Price, &p.Category, &p.IsActive,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan products: %w", err)
		}
		products = append(products, p)
	}

	return products, nil
}

func (r *postgresProductRepo) FindByID(ctx context.Context, id string) (*domain.Product, error) {
	query := `
		SELECT id, sku, name, description, price, category, is_active, created_at, updated_at
		FROM products
		WHERE id = $1
	`

	p := &domain.Product{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description,
		&p.Price, &p.Category, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("find product by id: %w", err)
	}

	return p, nil
}

func (r *postgresProductRepo) FindBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	query := `
		SELECT id, sku, name, description, price, category, is_active, created_at, updated_at
		FROM products
		WHERE sku = $1
	`

	p := &domain.Product{}
	err := r.db.QueryRow(ctx, query, sku).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description,
		&p.Price, &p.Category, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("find product by sku: %w", err)
	}

	return p, nil
}

// ── Stock Repository ──────────────────────────────────────────────────────────

type postgresStockRepo struct {
	db *pgxpool.Pool
}

// NewPostgresStockRepo creates a new Postgres stock repository.
func NewPostgresStockRepo(db *pgxpool.Pool) StockRepository {
	return &postgresStockRepo{db: db}
}

func (r *postgresStockRepo) GetByProductID(ctx context.Context, productID string) (*domain.StockLevel, error) {
	query := `
		SELECT id, product_id, quantity, reserved, warehouse_id, updated_at
		FROM stock_levels
		WHERE product_id = $1
	`

	s := &domain.StockLevel{}
	err := r.db.QueryRow(ctx, query, productID).Scan(
		&s.ID, &s.ProductID, &s.Quantity,
		&s.Reserved, &s.WarehouseID, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("get stock by product id: %w", err)
	}

	return s, nil
}

// Reserve atomically reserves stock using SELECT FOR UPDATE.
//
// Why SELECT FOR UPDATE?
// Without it: 100 concurrent requests could all read quantity=1,
// all see stock available, all decrement — stock goes to -99.
// 99 customers receive confirmation but no product.
//
// With SELECT FOR UPDATE:
// Only ONE transaction can lock the row at a time.
// The first transaction locks, checks, decrements, commits.
// Other 99 transactions wait, then read updated quantity=0,
// and return ErrInsufficientStock.
//
// This is a Postgres row-level lock — it only blocks other transactions
// trying to access THIS product's stock row. Other products are unaffected.
func (r *postgresStockRepo) Reserve(ctx context.Context, productID string, quantity int) error {
	// BeginTx starts a database transaction.
	// SELECT FOR UPDATE ONLY works inside a transaction.
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	// defer Rollback — runs automatically if we return before Commit.
	// If Commit() already ran successfully, Rollback() is a safe no-op.
	defer func() { _ = tx.Rollback(ctx) }()

	// Other transactions trying this query WAIT here until we COMMIT.
	// This serialises concurrent access to this product's stock row.
	var currentQuantity, currentReserved int
	err = tx.QueryRow(ctx, `
		SELECT quantity, reserved
		FROM stock_levels
		WHERE product_id = $1
		FOR UPDATE
	`, productID).Scan(&currentQuantity, &currentReserved)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrProductNotFound
		}
		return fmt.Errorf("lock stock row: %w", err)
	}

	// Check availability AFTER acquiring the lock.
	// available = quantity - reserved
	available := currentQuantity - currentReserved
	if available < quantity {
		// Not enough stock — Rollback releases the lock immediately
		return domain.ErrInsufficientStock
	}

	// Increment reserved count — units are now locked for this order.
	_, err = tx.Exec(ctx, `
		UPDATE stock_levels
		SET reserved = reserved + $1
		WHERE product_id = $2
	`, quantity, productID)
	if err != nil {
		return fmt.Errorf("update reserved: %w", err)
	}

	// COMMIT releases the lock — other waiting transactions can now proceed.
	// They will read the updated reserved count and check availability again.
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit reservation: %w", err)
	}

	return nil
}

// Release returns reserved units back to available.
// Called when an order is cancelled or payment fails.
func (r *postgresStockRepo) Release(ctx context.Context, productID string, quantity int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock the row before modifying reserved count
	var currentReserved int
	err = tx.QueryRow(ctx, `
		SELECT reserved FROM stock_levels
		WHERE product_id = $1
		FOR UPDATE
	`, productID).Scan(&currentReserved)
	if err != nil {
		return fmt.Errorf("lock stock row for release: %w", err)
	}

	// Prevent reserved from going below zero
	// (defensive check — shouldn't happen in normal flow)
	releaseQty := quantity
	if releaseQty > currentReserved {
		releaseQty = currentReserved
	}

	_, err = tx.Exec(ctx, `
		UPDATE stock_levels
		SET reserved = reserved - $1
		WHERE product_id = $2
	`, releaseQty, productID)
	if err != nil {
		return fmt.Errorf("update reserved on release: %w", err)
	}

	return tx.Commit(ctx)
}

// Adjust sets the total quantity for a product.
// Used when new stock arrives at the warehouse.
func (r *postgresStockRepo) Adjust(ctx context.Context, productID string, quantity int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE stock_levels
		SET quantity = $1
		WHERE product_id = $2
	`, quantity, productID)
	if err != nil {
		return fmt.Errorf("adjust stock: %w", err)
	}
	return nil
}

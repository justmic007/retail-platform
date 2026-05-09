// Package client — HTTP client for calling Inventory Service.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"retail-platform/order/internal/config"
	pkgerrors "retail-platform/pkg/errors"
)

// InventoryClient defines the interface for calling Inventory Service.
// The worker and service depend on this interface — not the concrete HTTP client.
// This allows tests to inject a mock and makes swapping HTTP for gRPC trivial.
type InventoryClientInterface interface {
	GetProduct(ctx context.Context, productID string) (*ProductResponse, error)
	Reserve(ctx context.Context, productID string, quantity int, orderID string) error
	Release(ctx context.Context, productID string, quantity int, orderID string) error
}

// ProductResponse is the relevant part of Inventory Service's product response.
type ProductResponse struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// reserveRequest is the body for POST /inventory/reserve.
type reserveRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	OrderID   string `json:"order_id"`
}

// releaseRequest is the body for POST /inventory/release.
type releaseRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	OrderID   string `json:"order_id"`
}

// InventoryClient calls Inventory Service on behalf of Order Service.
// Every request includes a JWT from ServiceTokenManager.
type InventoryClient struct {
	baseURL      string
	timeout      time.Duration
	tokenManager *ServiceTokenManager
	client       *http.Client
}

// NewInventoryClient creates a new InventoryClient.
func NewInventoryClient(cfg *config.Config, tokenManager *ServiceTokenManager) InventoryClientInterface {
	return &InventoryClient{
		baseURL:      cfg.InventoryServiceURL,
		timeout:      cfg.InventoryClientTimeout,
		tokenManager: tokenManager,
		client:       &http.Client{},
	}
}

// GetProduct fetches a product's name and price from Inventory Service.
// Called by the worker processor to get authoritative price at order time.
func (c *InventoryClient) GetProduct(ctx context.Context, productID string) (*ProductResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get service token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/products/"+productID, nil)
	if err != nil {
		return nil, fmt.Errorf("create get product request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get product request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, pkgerrors.NewNotFound("product", nil)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get product failed with status %d", resp.StatusCode)
	}

	var result struct {
		Product ProductResponse `json:"product"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode product response: %w", err)
	}

	return &result.Product, nil
}

// Reserve locks N units of a product for an order.
// Called by the worker processor after fetching prices.
func (c *InventoryClient) Reserve(ctx context.Context, productID string, quantity int, orderID string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("get service token: %w", err)
	}

	body, err := json.Marshal(reserveRequest{
		ProductID: productID,
		Quantity:  quantity,
		OrderID:   orderID,
	})
	if err != nil {
		return fmt.Errorf("marshal reserve request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/inventory/reserve", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create reserve request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("reserve request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return pkgerrors.NewInsufficientStock(productID)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("reserve failed with status %d", resp.StatusCode)
	}

	return nil
}

// Release returns previously reserved units back to available.
// Called when order processing fails or order is cancelled.
func (c *InventoryClient) Release(ctx context.Context, productID string, quantity int, orderID string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("get service token: %w", err)
	}

	body, err := json.Marshal(releaseRequest{
		ProductID: productID,
		Quantity:  quantity,
		OrderID:   orderID,
	})
	if err != nil {
		return fmt.Errorf("marshal release request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/inventory/release", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create release request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("release request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("release failed with status %d", resp.StatusCode)
	}

	return nil
}

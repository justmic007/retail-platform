// Package events defines the event types and shared channel used for
// async communication between services — specifically from Order Service
// to Notification Service.
//
// Why a Go channel instead of Kafka here?
// For this system, a buffered channel gives us async, non-blocking event
// dispatch within the same process. In a true multi-process deployment,
// you'd swap this for Kafka/RabbitMQ — but the dispatcher code in
// notification service wouldn't change much. This is intentional.
package events

import "time"

// EventType identifies what kind of event occurred.
type EventType string

const (
	EventOrderConfirmed EventType = "ORDER_CONFIRMED"
	EventOrderFailed    EventType = "ORDER_FAILED"
	EventOrderCancelled EventType = "ORDER_CANCELLED"
	EventStockLow       EventType = "STOCK_LOW"
)

// OrderEvent carries all the data a notification handler needs.
// Notice: no database IDs from internal tables — only what's needed
// to send a notification. Services share events, not database schemas.
type OrderEvent struct {
	Type      EventType `json:"type"`
	OrderID   string    `json:"order_id"`
	UserID    string    `json:"user_id"`
	UserEmail string    `json:"user_email"`
	Total     float64   `json:"total"`
	OccurredAt time.Time `json:"occurred_at"`
}

// StockEvent is published when stock drops below a threshold.
type StockEvent struct {
	Type       EventType `json:"type"`
	ProductID  string    `json:"product_id"`
	ProductName string   `json:"product_name"`
	StockLevel int       `json:"stock_level"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Bus holds the channels that services write to and read from.
// It is initialised once in main.go and passed down via dependency injection.
//
// Why buffered channels?
// A buffered channel (make(chan T, 100)) means the sender (Order Service)
// doesn't block if the receiver (Notification Service) is temporarily slow.
// Orders keep processing even if notifications are delayed.
// An unbuffered channel would block order processing until notifications are sent — bad.
type Bus struct {
	Orders chan OrderEvent // Order service writes, Notification service reads
	Stock  chan StockEvent // Inventory service writes, Notification service reads
}

// NewBus creates the event bus with sensible buffer sizes.
// Buffer size 100 means up to 100 events can be queued before the sender blocks.
func NewBus() *Bus {
	return &Bus{
		Orders: make(chan OrderEvent, 100),
		Stock:  make(chan StockEvent, 50),
	}
}

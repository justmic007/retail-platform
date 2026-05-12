// Package domain contains the pure business types for the Notification Service.
package domain

// NotificationType represents the type of notification to be sent.
type NotificationType string

const (
	NotificationOrderConfirmation NotificationType = "ORDER_CONFIRMATION"
	NotificationOrderFailed       NotificationType = "ORDER_FAILED"
	NotificationLowStock          NotificationType = "LOW_STOCK_ALERT"
	NotificationOrderCancelled    NotificationType = "ORDER_CANCELLED"
)

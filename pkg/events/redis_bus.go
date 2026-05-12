package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	channelOrders = "events:orders"
	channelStock  = "events:stock"
)

// RedisBus implements Publisher and Subscriber using Redis Pub/Sub.
type RedisBus struct {
	client *redis.Client
}

// NewRedisBus creates a new RedisBus.
func NewRedisBus(redisURL string) (*RedisBus, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	client := redis.NewClient(opts)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &RedisBus{client: client}, nil
}

// PublishOrder publishes an OrderEvent to Redis.
func (r *RedisBus) PublishOrder(ctx context.Context, event OrderEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal order event: %w", err)
	}
	return r.client.Publish(ctx, channelOrders, data).Err()
}

// PublishStock publishes a StockEvent to Redis.
func (r *RedisBus) PublishStock(ctx context.Context, event StockEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal stock event: %w", err)
	}
	return r.client.Publish(ctx, channelStock, data).Err()
}

// SubscribeOrders returns a channel of OrderEvents from Redis.
func (r *RedisBus) SubscribeOrders(ctx context.Context) (<-chan OrderEvent, error) {
	sub := r.client.Subscribe(ctx, channelOrders)
	out := make(chan OrderEvent, 100)

	go func() {
		defer close(out)
		for {
			msg, err := sub.ReceiveMessage(ctx)
			if err != nil {
				return
			}
			var event OrderEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				continue
			}
			select {
			case out <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

// SubscribeStock returns a channel of StockEvents from Redis.
func (r *RedisBus) SubscribeStock(ctx context.Context) (<-chan StockEvent, error) {
	sub := r.client.Subscribe(ctx, channelStock)
	out := make(chan StockEvent, 50)

	go func() {
		defer close(out)
		for {
			msg, err := sub.ReceiveMessage(ctx)
			if err != nil {
				return
			}
			var event StockEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				continue
			}
			select {
			case out <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

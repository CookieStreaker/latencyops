package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"latencyops/internal/domain"
)

type StateRepository interface {
	SavePingState(ctx context.Context, result domain.PingResult) error
	IncrementFailureCount(ctx context.Context, endpointID string) (int64, error)
	ResetFailureCount(ctx context.Context, endpointID string) error
	PublishPingResult(ctx context.Context, result domain.PingResult) error
}

type redisStateRepo struct {
	client *redis.Client
}

func NewRedisStateRepo(client *redis.Client) StateRepository {
	return &redisStateRepo{client: client}
}

// SavePingState writes the sub-millisecond evaluation status to Redis for the UI dashboard.
func (r *redisStateRepo) SavePingState(ctx context.Context, result domain.PingResult) error {
	key := fmt.Sprintf("endpoint:%s:state", result.EndpointID)
	
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal ping result for redis: %w", err)
	}

	// Keep state cached for 24 hours
	if err := r.client.Set(ctx, key, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to save ping state to redis: %w", err)
	}

	return nil
}

// IncrementFailureCount implements the sliding-window error counter for rate limits and threshold breaches.
func (r *redisStateRepo) IncrementFailureCount(ctx context.Context, endpointID string) (int64, error) {
	key := fmt.Sprintf("endpoint:%s:failures", endpointID)
	
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment failure count: %w", err)
	}

	// Set expiration on the counter so it acts as a sliding window (e.g., 5 minutes)
	if count == 1 {
		r.client.Expire(ctx, key, 5*time.Minute)
	}

	return count, nil
}

func (r *redisStateRepo) ResetFailureCount(ctx context.Context, endpointID string) error {
	key := fmt.Sprintf("endpoint:%s:failures", endpointID)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to reset failure count: %w", err)
	}
	return nil
}

// PublishPingResult streams the result to a Pub/Sub channel so our Webhook dispatcher can process it immediately.
func (r *redisStateRepo) PublishPingResult(ctx context.Context, result domain.PingResult) error {
	channel := "ping_results_stream"
	
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal ping result for pubsub: %w", err)
	}

	if err := r.client.Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish ping result to stream: %w", err)
	}

	return nil
}
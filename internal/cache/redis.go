package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the Redis client
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client connection
func NewRedisClient(ctx context.Context, redisURL string) (*RedisClient, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		// Fallback: try prepending redis:// for bare host:port format
		opts, err = redis.ParseURL(fmt.Sprintf("redis://%s", redisURL))
	}
	if err != nil {
		// Try as host:port format
		opts = &redis.Options{
			Addr: redisURL,
		}
	}

	client := redis.NewClient(opts)

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{client: client}, nil
}

// IsAvailable returns true if the Redis client is connected
func (r *RedisClient) IsAvailable() bool {
	return r != nil && r.client != nil
}

// Client returns the underlying Redis client
func (r *RedisClient) Client() *redis.Client {
	if r == nil {
		return nil
	}
	return r.client
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

// Get retrieves a value from cache
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	if r == nil || r.client == nil {
		return "", redis.Nil
	}
	return r.client.Get(ctx, key).Result()
}

// Set stores a value in cache with TTL
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Del deletes one or more keys
func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	if r == nil || r.client == nil {
		return false, nil
	}
	n, err := r.client.Exists(ctx, key).Result()
	return n > 0, err
}

// Incr increments a counter
func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	if r == nil || r.client == nil {
		return 0, nil
	}
	return r.client.Incr(ctx, key).Result()
}

// Expire sets expiration on a key
func (r *RedisClient) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Expire(ctx, key, ttl).Err()
}

// LPush prepends values to a list
func (r *RedisClient) LPush(ctx context.Context, key string, values ...interface{}) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.LPush(ctx, key, values...).Err()
}

// LTrim trims a list to the specified range
func (r *RedisClient) LTrim(ctx context.Context, key string, start, stop int64) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.LTrim(ctx, key, start, stop).Err()
}

// LRange returns a range of elements from a list
func (r *RedisClient) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	return r.client.LRange(ctx, key, start, stop).Result()
}

// DeleteByPattern deletes all keys matching a pattern
func (r *RedisClient) DeleteByPattern(ctx context.Context, pattern string) error {
	if r == nil || r.client == nil {
		return nil
	}
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) > 0 {
		return r.client.Del(ctx, keys...).Err()
	}
	return nil
}

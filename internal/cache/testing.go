package cache

import "github.com/redis/go-redis/v9"

// NewRedisClientFromClient creates a RedisClient from an existing redis.Client.
// This is used for testing with miniredis.
func NewRedisClientFromClient(client *redis.Client) *RedisClient {
	return &RedisClient{client: client}
}

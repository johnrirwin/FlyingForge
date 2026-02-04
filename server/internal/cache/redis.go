package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache is a Redis-backed cache implementation
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// RedisConfig holds configuration for the Redis cache
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	Prefix   string
}

// NewRedis creates a new Redis cache with the specified configuration
func NewRedis(cfg RedisConfig, ttl time.Duration) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "mcp-news:"
	}

	return &RedisCache{
		client: client,
		ttl:    ttl,
		prefix: prefix,
	}, nil
}

func (c *RedisCache) key(k string) string {
	return c.prefix + k
}

func (c *RedisCache) Get(key string) (interface{}, bool) {
	ctx := context.Background()

	data, err := c.client.Get(ctx, c.key(key)).Bytes()
	if err != nil {
		return nil, false
	}

	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, false
	}

	return value, true
}

func (c *RedisCache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

func (c *RedisCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	ctx := context.Background()

	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	c.client.Set(ctx, c.key(key), data, ttl)
}

func (c *RedisCache) Delete(key string) {
	ctx := context.Background()
	c.client.Del(ctx, c.key(key))
}

func (c *RedisCache) Clear() {
	ctx := context.Background()

	// Use SCAN to find all keys with our prefix and delete them
	iter := c.client.Scan(ctx, 0, c.prefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		c.client.Del(ctx, iter.Val())
	}
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// Client returns the underlying Redis client for use by other components
func (c *RedisCache) Client() *redis.Client {
	return c.client
}

// Ensure RedisCache implements Cache interface
var _ Cache = (*RedisCache)(nil)

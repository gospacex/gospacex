package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// RedisGenerator generates Redis database templates
type RedisGenerator struct {
	OutputDir string
}

// NewRedisGenerator creates a new RedisGenerator
func NewRedisGenerator(outputDir string) *RedisGenerator {
	return &RedisGenerator{
		OutputDir: outputDir,
	}
}

// Generate creates Redis template files
func (g *RedisGenerator) Generate() error {
	files := map[string]string{
		"config.go.tmpl": redisConfigTemplate,
		"client.go.tmpl": redisClientTemplate,
		"cache.go.tmpl":  redisCacheTemplate,
	}

	dir := filepath.Join(g.OutputDir, "templates", "database", "redis")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("write file %s: %w", name, err)
		}
	}

	return nil
}

const redisConfigTemplate = `package redis

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Config Redis configuration
type Config struct {
	Host     string ` + "`yaml:\"host\" env:\"REDIS_HOST\" default:\"localhost\"`" + `
	Port     int    ` + "`yaml:\"port\" env:\"REDIS_PORT\" default:\"6379\"`" + `
	Password string ` + "`yaml:\"password\" env:\"REDIS_PASSWORD\"`" + `
	DB       int    ` + "`yaml:\"db\" env:\"REDIS_DB\" default:\"0\"`" + `
	PoolSize int    ` + "`yaml:\"pool_size\" default:\"100\"`" + `
}

// NewClient creates a new Redis client
func NewClient(cfg *Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
}
`

const redisClientTemplate = `package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client Redis client wrapper
type Client struct {
	client *redis.Client
}

// NewClient creates a new Redis client
func NewClient(cfg *Config) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	return &Client{client: client}, nil
}

// Client returns the underlying redis.Client
func (c *Client) Client() *redis.Client {
	return c.client
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.client.Close()
}

// HealthCheck checks Redis connectivity
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
`

const redisCacheTemplate = `package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheService Redis cache service
type CacheService struct {
	client *redis.Client
}

// NewCacheService creates a new CacheService
func NewCacheService(client *redis.Client) *CacheService {
	return &CacheService{client: client}
}

// Set sets a key-value pair with expiration
func (c *CacheService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, expiration).Err()
}

// Get gets a value by key
func (c *CacheService) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Del deletes a key
func (c *CacheService) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Exists checks if a key exists
func (c *CacheService) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// HSet sets a hash field
func (c *CacheService) HSet(ctx context.Context, hash, key string, value interface{}) error {
	return c.client.HSet(ctx, hash, key, value).Err()
}

// HGet gets a hash field
func (c *CacheService) HGet(ctx context.Context, hash, key string) (string, error) {
	return c.client.HGet(ctx, hash, key).Result()
}
`

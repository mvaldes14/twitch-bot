// package cache provides a simple Redis-based caching service for storing and retrieving tokens.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

// Service struct
type Service struct {
	Log *telemetry.CustomLogger
}

// Token represents a generic token structure
type Token struct {
	Key        string
	Value      string
	Expiration time.Duration
}

var (
	ctx           = context.Background()
	rdb           *redis.Client
	errorNoToken  = errors.New("no token found for the given key")
	cacheInstance *Service
)

// NewCacheService initializes a new CacheService instance (singleton)
func NewCacheService() *Service {
	if cacheInstance != nil {
		return cacheInstance
	}

	redisURL := os.Getenv("REDIS_URL")

	logger := telemetry.NewLogger("cache")
	rdb = redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		panic("Could not connect to Redis: " + err.Error())
	}
	logger.Info("Connected to Redis successfully")
	cacheInstance = &Service{Log: logger}
	return cacheInstance
}

// GetToken retrieves a token from Redis
func (c *Service) GetToken(key string) (string, error) {
	_, span := telemetry.StartSpan(ctx, "redis.get_token",
		attribute.String("cache.key", key),
	)
	defer span.End()

	c.Log.Info(fmt.Sprintf("Retrieving token '%s' from Redis cache", key))
	val, err := rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		c.Log.Error(fmt.Sprintf("Token '%s' not found in Redis cache - application may not be properly initialized", key), errorNoToken)
		telemetry.RecordError(span, errorNoToken)
		telemetry.IncrementCacheOperation(ctx, "get", "miss")
		return "", err
	}
	if err != nil {
		c.Log.Error(fmt.Sprintf("Error retrieving token '%s' from Redis: %v", key, err), err)
		telemetry.RecordError(span, err)
		telemetry.IncrementCacheOperation(ctx, "get", "error")
		return "", err
	}
	var token Token
	if err := json.Unmarshal([]byte(val), &token); err != nil {
		c.Log.Error(fmt.Sprintf("Failed to unmarshal token '%s' from Redis", key), err)
		telemetry.RecordError(span, err)
		telemetry.IncrementCacheOperation(ctx, "get", "error")
		return "", err
	}
	c.Log.Info(fmt.Sprintf("Retrieved token '%s' from Redis cache", key))
	telemetry.IncrementCacheOperation(ctx, "get", "hit")
	return token.Value, nil
}

// StoreToken stores a key token in Redis
func (c *Service) StoreToken(tk Token) error {
	_, span := telemetry.StartSpan(ctx, "redis.store_token",
		attribute.String("cache.key", tk.Key),
		attribute.Int64("cache.expiration_seconds", int64(tk.Expiration.Seconds())),
	)
	defer span.End()

	c.Log.Info(fmt.Sprintf("Storing token '%s' in Redis with expiration %s", tk.Key, tk.Expiration))
	jsonToken, err := json.Marshal(tk)
	if err != nil {
		c.Log.Error(fmt.Sprintf("Failed to marshal token '%s': %v", tk.Key, err), err)
		telemetry.RecordError(span, err)
		return err
	}
	if err := rdb.Set(ctx, tk.Key, jsonToken, tk.Expiration).Err(); err != nil {
		c.Log.Error(fmt.Sprintf("Failed to store token '%s' in Redis: %v", tk.Key, err), err)
		telemetry.RecordError(span, err)
		telemetry.IncrementCacheOperation(ctx, "store", "error")
		return err
	}
	telemetry.IncrementCacheOperation(ctx, "store", "success")
	c.Log.Info(fmt.Sprintf("Token '%s' stored successfully in Redis", tk.Key))
	return nil
}

// DeleteToken removes a token from Redis
func (c *Service) DeleteToken(key string) error {
	_, span := telemetry.StartSpan(ctx, "redis.delete_token",
		attribute.String("cache.key", key),
	)
	defer span.End()

	c.Log.Info(fmt.Sprintf("Deleting token '%s' from Redis", key))
	if err := rdb.Del(ctx, key).Err(); err != nil {
		c.Log.Error(fmt.Sprintf("Failed to delete token '%s' from Redis: %v", key, err), err)
		telemetry.RecordError(span, err)
		telemetry.IncrementCacheOperation(ctx, "delete", "error")
		return err
	}
	telemetry.IncrementCacheOperation(ctx, "delete", "success")
	c.Log.Info(fmt.Sprintf("Token '%s' deleted successfully from Redis", key))
	return nil
}

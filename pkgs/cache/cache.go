// package cache provides a simple Redis-based caching service for storing and retrieving tokens.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mvaldes14/twitch-bot/pkgs/telemetry"
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
	ctx                   = context.Background()
	rdb                   *redis.Client
	errorFailedConnection = errors.New("failed to connect to Redis")
	errorNoToken          = errors.New("no token found for the given key")
	cacheInstance         *Service
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
	c.Log.Info("Retrieving token from Redis", key)
	val, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		c.Log.Error("Token not found in Redis", errorNoToken)
		return "", err
	}
	var token Token
	if err := json.Unmarshal([]byte(val), &token); err != nil {
		c.Log.Error("Failed to unmarshal token", err)
		return "", err
	}
	return token.Value, nil
}

// StoreToken stores a key token in Redis
func (c *Service) StoreToken(tk Token) error {
	c.Log.Info("Storing token in Redis", tk.Key)
	jsonToken, err := json.Marshal(tk)
	if err != nil {
		c.Log.Error("Failed to marshal token", err)
		return err
	}
	if err := rdb.Set(ctx, tk.Key, jsonToken, tk.Expiration).Err(); err != nil {
		c.Log.Error("Failed to store token in Redis", err)
		return err
	}
	c.Log.Info("Token stored successfully", tk.Key)
	return nil
}

// DeleteToken removes a token from Redis
func (c *Service) DeleteToken(key string) error {
	c.Log.Info("Deleting token from Redis", key)
	if err := rdb.Del(ctx, key).Err(); err != nil {
		c.Log.Error("Failed to delete token from Redis", err)
		return err
	}
	c.Log.Info("Token deleted successfully", key)
	return nil
}

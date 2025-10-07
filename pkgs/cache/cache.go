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

// CacheService handles caching operations
type CacheService struct {
	Logger *telemetry.BotLogger
}

// Cache interface defines methods for token management
type Cache interface {
	GetToken(key string) (string, error)
	StoreToken(tk Token) error
	DeleteToken(key string) error
}

// Token represents a generic token structure
type Token struct {
	Key        string
	Value      string
	Expiration time.Duration
}

// TODO: Think of all the possible errors we can throw based on the service
var (
	ctx                    = context.Background()
	rdb                    *redis.Client
	errorNoToken           = errors.New("Could not find the token")
	errorNoRedisConnection = errors.New("Could not connect to redis")
	cacheInstance          *CacheService
)

// NewCacheService initializes a new CacheService instance (singleton)
func NewCacheService() *CacheService {
	if cacheInstance != nil {
		return cacheInstance
	}

	redisURL := os.Getenv("REDIS_URL")

	logger := telemetry.NewLogger("cache")
	rdb = redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		panic(errorNoRedisConnection)
	}
	cacheInstance = &CacheService{Logger: logger}
	return cacheInstance
}

// GetToken retrieves a token from Redis
func (c *CacheService) GetToken(key string) (Token, error) {
	c.Logger.Info("Retrieving token from Redis:" + key)
	var token Token
	val, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		c.Logger.Error(errorNoToken)
		return token, err
	}
	if err := json.Unmarshal([]byte(val), &token); err != nil {
		c.Logger.Error(err)
		return token, err
	}
	return token, nil
}

// StoreToken stores a key token in Redis
func (c *CacheService) StoreToken(tk Token) error {
	c.Logger.Info("Storing token in Redis: " + tk.Key)
	jsonToken, err := json.Marshal(tk)
	if err != nil {
		c.Logger.Error(err)
		return err
	}
	if err := rdb.Set(ctx, tk.Key, jsonToken, tk.Expiration).Err(); err != nil {
		c.Logger.Error(err)
		return err
	}
	c.Logger.Info("Token stored successfully: " + tk.Key)
	return nil
}

// DeleteToken removes a token from Redis
func (c *CacheService) DeleteToken(key string) error {
	c.Logger.Info("Deleting token from Redis: " + key)
	if err := rdb.Del(ctx, key).Err(); err != nil {
		c.Logger.Error(err)
		return err
	}
	c.Logger.Info("Token deleted successfully: " + key)
	return nil
}

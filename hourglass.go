package hourglass

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed consume.lua
var consumeScriptData string

type Config struct {
	RedisAddress  string         `json:"redisAddress"`
	RedisPassword string         `json:"redisPassword"`
	Limits        map[string]int `json:"limits"`
	PoolSize      int            `json:"poolSize"`
	MinIdleConns  int            `json:"minIdleConns"`
	MaxRetries    int            `json:"maxRetries"`
	DialTimeout   time.Duration  `json:"dialTimeout"`
	ReadTimeout   time.Duration  `json:"readTimeout"`
	WriteTimeout  time.Duration  `json:"writeTimeout"`
	PoolTimeout   time.Duration  `json:"poolTimeout"`
	IdleTimeout   time.Duration  `json:"idleTimeout"`
	MaxConnAge    time.Duration  `json:"maxConnAge"`
}

type HourGlass struct {
	appConfig     Config
	redisClient   *redis.Client
	consumeScript *redis.Script
}

func New(config *Config) (*HourGlass, error) {
	// Set defaults for connection pooling
	if config.PoolSize == 0 {
		config.PoolSize = 10
	}
	if config.MinIdleConns == 0 {
		config.MinIdleConns = 5
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.DialTimeout == 0 {
		config.DialTimeout = 5 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 3 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 3 * time.Second
	}
	if config.PoolTimeout == 0 {
		config.PoolTimeout = 4 * time.Second
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 5 * time.Minute
	}
	if config.MaxConnAge == 0 {
		config.MaxConnAge = 30 * time.Minute
	}

	// Connect to Redis with optimized connection pool settings
	rdb := redis.NewClient(&redis.Options{
		Addr:            config.RedisAddress,
		Password:        config.RedisPassword,
		DB:              0,
		PoolSize:        config.PoolSize,
		MinIdleConns:    config.MinIdleConns,
		MaxRetries:      config.MaxRetries,
		DialTimeout:     config.DialTimeout,
		ReadTimeout:     config.ReadTimeout,
		WriteTimeout:    config.WriteTimeout,
		PoolTimeout:     config.PoolTimeout,
		ConnMaxIdleTime: config.IdleTimeout,
		ConnMaxLifetime: config.MaxConnAge,
	})

	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}

	consumeScript := redis.NewScript(consumeScriptData)

	return &HourGlass{
		appConfig:     *config,
		redisClient:   rdb,
		consumeScript: consumeScript,
	}, nil
}

func getKey(featureName, username string) string {
	return fmt.Sprintf("%s:%s:%s", featureName, username, time.Now().UTC().Format("2006-01-02"))
}

func (hg *HourGlass) Get(ctx context.Context, featureName, userName string) (current int, limit int) {

	limit, exists := hg.appConfig.Limits[featureName]
	if !exists {
		return -1, -1
	}

	cmd := hg.redisClient.Get(ctx, getKey(featureName, userName))
	if cmd.Err() != nil {
		return -1, limit
	}

	consumed, err := cmd.Int()
	if err != nil {
		return -1, limit
	}

	return consumed, limit
}

func (hg *HourGlass) Consume(ctx context.Context, featureName, userName string) (current int, limit int, can bool) {
	key := getKey(featureName, userName)
	limit, exists := hg.appConfig.Limits[featureName]
	if !exists {
		return -1, -1, true
	}

	// Calculate TTL until end of day
	ttl := int(timeUntilEndOfDay().Seconds())

	result := hg.consumeScript.Run(ctx, hg.redisClient, []string{key}, limit, ttl)
	if result.Err() != nil {
		// Fail open
		return -1, limit, true
	}

	resultArray := result.Val().([]interface{})
	current = int(resultArray[0].(int64))
	limit = int(resultArray[1].(int64))
	can = resultArray[2].(int64) == 1

	return current, limit, can
}

func (hg *HourGlass) Credit(ctx context.Context, featureName, userName string) (current int, limit int) {
	key := fmt.Sprintf("%s:%s:%s", featureName, userName, time.Now().UTC().Format("2006-01-02"))

	limit, exists := hg.appConfig.Limits[featureName]
	if !exists {
		return -1, -1
	}

	cmd := hg.redisClient.Decr(ctx, key)
	if cmd.Err() != nil {
		return -1, limit
	}

	return int(cmd.Val()), hg.appConfig.Limits[featureName]
}

func (hg *HourGlass) Close() error {
	return hg.redisClient.Close()
}

func timeUntilEndOfDay() time.Duration {
	timeRightNow := time.Now().UTC()
	endOfDay := time.Date(timeRightNow.Year(), timeRightNow.Month(), timeRightNow.Day()+1, 0, 0, 0, 0, time.UTC)
	duration := endOfDay.Sub(timeRightNow)
	return duration
}

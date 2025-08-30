# HourGlass ⏳

A high-performance, light-weight, Redis-based rate limiting library for Go that provides daily usage quotas per user and feature. 

## Features

- **Daily Rate Limiting**: Automatic daily quota reset with precise time calculations
- **Multi-Feature Support**: Different limits for different features in applications
- **User-Scoped**: Individual quotas per user for each feature
- **Atomic Operations**: Redis Lua scripts ensure consistency under high concurrency
- **Connection Pooling**: Optimized Redis connection management for high throughput
- **Fail-Open Policy**: Gracefully handles Redis unavailability
- **Credit System**: Support for refunding consumed quotas

## Quick Start

### Installation

```bash
go get github.com/pyljain/hourglass
```

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"
    
    "hourglass"
)

func main() {
    ctx := context.Background()
    
    // Configure HourGlass
    cfg := &hourglass.Config{
        RedisAddress: "localhost:6379",
        Limits: map[string]int{
            "api-calls":    1000,
            "file-uploads": 50,
            "ai-requests":  10,
        },
    }
    
    hg, err := hourglass.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer hg.Close()
    
    // Check current usage
    current, limit := hg.Get(ctx, "api-calls", "user123")
    log.Printf("User has used %d/%d API calls today", current, limit)
    
    // Attempt to consume quota
    current, limit, allowed := hg.Consume(ctx, "api-calls", "user123")
    if !allowed {
        log.Printf("Rate limit exceeded: %d/%d", current, limit)
        return
    }
    
    // Process the request...
    log.Printf("Request processed. Usage: %d/%d", current, limit)
    
    // If operation failed, credit back the quota
    // current, limit = hg.Credit(ctx, "api-calls", "user123")
}
```

## Configuration

### Basic Configuration

```go
type Config struct {
    RedisAddress  string         `json:"redisAddress"`
    RedisPassword string         `json:"redisPassword"`
    Limits        map[string]int `json:"limits"`
}
```

### Advanced Connection Pool Configuration

```go
cfg := &hourglass.Config{
    RedisAddress: "localhost:6379",
    Limits: map[string]int{
        "feature1": 100,
    },
    
    // Connection Pool Settings
    PoolSize:     15,                  // Max connections
    MinIdleConns: 8,                   // Min idle connections
    MaxRetries:   3,                   // Retry failed operations
    DialTimeout:  5 * time.Second,     // Connection timeout
    ReadTimeout:  2 * time.Second,     // Read operation timeout
    WriteTimeout: 2 * time.Second,     // Write operation timeout
    PoolTimeout:  3 * time.Second,     // Wait for connection from pool
}
```

## API Reference

### Methods

#### `New(config *Config) (*HourGlass, error)`
Creates a new HourGlass instance with the provided configuration.

#### `Get(ctx context.Context, featureName, userName string) (current int, limit int)`
Retrieves the current usage count for a user and feature without consuming quota.

#### `Consume(ctx context.Context, featureName, userName string) (current int, limit int, can bool)`
Attempts to consume one unit of quota. Returns the updated count, limit, and whether the operation was allowed.

#### `Credit(ctx context.Context, featureName, userName string) (current int, limit int)`
Returns one unit of quota back to the user (useful for failed operations).

#### `Close() error`
Closes the Redis connection pool.

## Key Design Decisions

### Daily Reset Strategy
- Uses UTC time for consistent daily boundaries
- Keys format: `feature:user:YYYY-MM-DD`
- Automatic expiration at end of day using Redis TTL

### Atomic Operations
- Lua scripts ensure race-condition-free quota consumption
- Handles edge cases like concurrent access and quota overflow

### Fail-Open Policy
- If Redis is unavailable, `Consume()` allows the operation
- Prevents total service disruption during Redis outages

## Performance Characteristics

- **Throughput**: >50K operations/second with proper connection pooling
- **Latency**: Sub-millisecond for local Redis, <5ms for remote
- **Memory**: Minimal allocations in hot path
- **Concurrency**: Thread-safe, supports high concurrent access

## Redis Requirements

- **Version**: Redis 3.2+ (for Lua script support)
- **Memory**: ~100 bytes per user-feature-day combination
- **Network**: Low latency connection recommended for best performance

## Error Handling

HourGlass follows a fail-open philosophy:
- Redis connection errors allow operations to proceed
- Invalid configurations return errors during initialization
- Malformed responses are treated as quota available

## Best Practices

1. **Connection Pooling**: Configure appropriate pool sizes based on your load
2. **Context Timeouts**: Always use contexts with reasonable timeouts
3. **Error Monitoring**: Monitor Redis connection health and error rates
4. **Quota Planning**: Set limits based on your system's capacity, not user desires
5. **Credit Operations**: Always credit back quotas for failed operations

## Examples

See the [examples/](examples/) directory for complete usage examples including:
- Basic rate limiting
- Error handling patterns  
- Connection pool optimization
- Monitoring and metrics integration

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

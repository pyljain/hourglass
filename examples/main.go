package main

import (
	"context"
	"hourglass"
	"log"
	"os"
	"time"
)

func main() {
	ctx := context.Background()
	cfg := &hourglass.Config{
		RedisAddress: "localhost:6379",
		Limits: map[string]int{
			"lattice":     4,
			"claude-code": 10,
			"agentic":     10,
		},
		PoolSize:     15,                  // Increased for higher concurrency
		MinIdleConns: 8,                   // Keep more idle connections ready
		MaxRetries:   3,                   // Retry failed operations
		DialTimeout:  5 * time.Second,     // Timeout for establishing connections
		ReadTimeout:  2 * time.Second,     // Faster read timeout
		WriteTimeout: 2 * time.Second,     // Faster write timeout
		PoolTimeout:  3 * time.Second,     // Timeout waiting for connection from pool
	}

	hg, err := hourglass.New(cfg)
	if err != nil {
		log.Fatalln(err)
	}
	defer hg.Close()

	// Check current value for a feature - UI to display the amount
	current, limit := hg.Get(ctx, "lattice", "pj11993")
	log.Printf("Current: %d, Limit: %d", current, limit)

	// Increment value for a feature for a user
	current, limit, isAllowed := hg.Consume(ctx, "lattice", "pj11993")
	if !isAllowed {
		// Throw error to user saying limit exceeded
		log.Printf("Not allowed")
		os.Exit(-1)
	}
	log.Printf("After increment Current: %d, Limit: %d", current, limit)

	// Run logic - run lattice
	log.Printf("Run logic")

	// If lattice fails then return credits
	// current, limit = hg.Credit(ctx, "lattice", "pj11993")
}

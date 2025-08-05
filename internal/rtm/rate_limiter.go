// Package rtm provides RTM API integration with rate limiting
package rtm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter implements RTM API rate limiting with burst support.
// RTM allows 1 request/second average with 3 burst capacity.
type RateLimiter struct {
	mu           sync.Mutex
	tokens       float64
	lastRefill   time.Time
	maxTokens    float64
	refillRate   float64 // tokens per second
	backoffUntil time.Time
	metrics      *RateLimitMetrics
}

// RateLimitMetrics tracks rate limiter performance
type RateLimitMetrics struct {
	mu              sync.RWMutex
	RequestsTotal   int64
	RequestsBlocked int64
	Errors503       int64
	BurstUsed       int64
	AvgWaitTime     time.Duration
	totalWaitTime   time.Duration
	waitTimeCount   int64
}

// RateLimitMetricsSnapshot is a copy of metrics without the mutex
type RateLimitMetricsSnapshot struct {
	RequestsTotal   int64
	RequestsBlocked int64
	Errors503       int64
	BurstUsed       int64
	AvgWaitTime     time.Duration
}

// NewRateLimiter creates a rate limiter for RTM API
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		tokens:     3.0, // Start with full burst capacity
		maxTokens:  3.0, // RTM allows 3 burst
		refillRate: 1.0, // 1 request per second
		lastRefill: time.Now(),
		metrics:    &RateLimitMetrics{},
	}
}

// Wait blocks until a request slot is available or context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context) error {
	startWait := time.Now()
	defer func() {
		rl.recordWaitTime(time.Since(startWait))
	}()

	for {
		rl.mu.Lock()

		// Check if we're in backoff period (after 503 error)
		if time.Now().Before(rl.backoffUntil) {
			backoffDuration := time.Until(rl.backoffUntil)
			rl.mu.Unlock()

			select {
			case <-time.After(backoffDuration):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Refill tokens based on elapsed time
		rl.refillTokens()

		// Check if we have a token available
		if rl.tokens >= 1.0 {
			rl.tokens--
			rl.metrics.RequestsTotal++
			if rl.tokens < 2.0 { // Used some burst capacity
				rl.metrics.BurstUsed++
			}
			rl.mu.Unlock()
			return nil
		}

		// Calculate wait time until next token
		timeToNextToken := time.Second / time.Duration(rl.refillRate)
		rl.metrics.RequestsBlocked++
		rl.mu.Unlock()

		select {
		case <-time.After(timeToNextToken):
			// Continue loop to try again
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// refillTokens adds tokens based on elapsed time (must be called with lock held)
func (rl *RateLimiter) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()

	// Add tokens based on elapsed time
	rl.tokens += elapsed * rl.refillRate

	// Cap at max tokens (burst capacity)
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}

	rl.lastRefill = now
}

// HandleError503 implements exponential backoff for rate limit errors
func (rl *RateLimiter) HandleError503() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.metrics.Errors503++

	// Exponential backoff: 2^n seconds where n is consecutive 503s
	// Start with 2 seconds, max 60 seconds
	backoffSeconds := 2.0
	if rl.metrics.Errors503 > 1 {
		shiftAmount := min(int(rl.metrics.Errors503), 6)
		backoffSeconds = float64(uint(1) << uint(shiftAmount))
	}

	rl.backoffUntil = time.Now().Add(time.Duration(backoffSeconds) * time.Second)
	rl.tokens = 0 // Clear tokens to force wait
}

// ResetBackoff clears backoff after successful request
func (rl *RateLimiter) ResetBackoff() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.metrics.Errors503 = 0
	rl.backoffUntil = time.Time{}
}

// GetMetrics returns current rate limiter metrics
func (rl *RateLimiter) GetMetrics() RateLimitMetricsSnapshot {
	rl.metrics.mu.RLock()
	defer rl.metrics.mu.RUnlock()

	metrics := RateLimitMetricsSnapshot{
		RequestsTotal:   rl.metrics.RequestsTotal,
		RequestsBlocked: rl.metrics.RequestsBlocked,
		Errors503:       rl.metrics.Errors503,
		BurstUsed:       rl.metrics.BurstUsed,
	}
	if rl.metrics.waitTimeCount > 0 {
		metrics.AvgWaitTime = rl.metrics.totalWaitTime / time.Duration(rl.metrics.waitTimeCount)
	}
	return metrics
}

// recordWaitTime tracks wait time metrics
func (rl *RateLimiter) recordWaitTime(duration time.Duration) {
	rl.metrics.mu.Lock()
	defer rl.metrics.mu.Unlock()

	rl.metrics.totalWaitTime += duration
	rl.metrics.waitTimeCount++
}

// EstimateDuration estimates time for N operations
func (rl *RateLimiter) EstimateDuration(operations int) time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Account for current tokens (burst)
	rl.refillTokens()
	availableNow := int(rl.tokens)

	if operations <= availableNow {
		return 100 * time.Millisecond // Nearly instant with burst
	}

	// Remaining operations need to wait for refill
	remaining := operations - availableNow
	return time.Duration(remaining) * time.Second
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// WaitForBatch optimizes waiting for batch operations
// Returns a channel that sends when ready for next request
func (rl *RateLimiter) WaitForBatch(ctx context.Context, total int) <-chan error {
	errChan := make(chan error, 1)

	go func() {
		defer close(errChan)

		for i := 0; i < total; i++ {
			if err := rl.Wait(ctx); err != nil {
				errChan <- fmt.Errorf("rate limit wait cancelled at item %d/%d: %w", i+1, total, err)
				return
			}

			// Send nil to signal ready for next request
			select {
			case errChan <- nil:
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
		}
	}()

	return errChan
}

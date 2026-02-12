package retry

import (
	"context"
	"fmt"
	"time"
)

// Config holds retry configuration
type Config struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	MaxTotalTimeout time.Duration
}

// DefaultConfig returns a default retry configuration with 1 minute max timeout
func DefaultConfig() Config {
	return Config{
		MaxAttempts:     10,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		BackoffFactor:   2.0,
		MaxTotalTimeout: 60 * time.Second, // 1 minute max
	}
}

// Do executes the given function with exponential backoff retry logic
func Do(ctx context.Context, cfg Config, fn func() error) error {
	if cfg.MaxTotalTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.MaxTotalTimeout)
		defer cancel()
	}

	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("retry aborted after %d attempts: %w (last error: %v)", attempt-1, ctx.Err(), lastErr)
			}
			return fmt.Errorf("retry aborted: %w", ctx.Err())
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if attempt == cfg.MaxAttempts {
			return fmt.Errorf("max retry attempts (%d) exceeded: %w", cfg.MaxAttempts, lastErr)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("retry aborted after %d attempts: %w (last error: %v)", attempt, ctx.Err(), lastErr)
		case <-time.After(delay):
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * cfg.BackoffFactor)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return fmt.Errorf("max retry attempts exceeded: %w", lastErr)
}

// DoWithLog executes the function with retry and logs each attempt
func DoWithLog(ctx context.Context, cfg Config, serviceName string, fn func() error, logFn func(attempt int, err error, nextDelay time.Duration)) error {
	if cfg.MaxTotalTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.MaxTotalTimeout)
		defer cancel()
	}

	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("%s: retry aborted after %d attempts: %w (last error: %v)", serviceName, attempt-1, ctx.Err(), lastErr)
			}
			return fmt.Errorf("%s: retry aborted: %w", serviceName, ctx.Err())
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if attempt == cfg.MaxAttempts {
			return fmt.Errorf("%s: max retry attempts (%d) exceeded: %w", serviceName, cfg.MaxAttempts, lastErr)
		}

		if logFn != nil {
			logFn(attempt, err, delay)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("%s: retry aborted after %d attempts: %w (last error: %v)", serviceName, attempt, ctx.Err(), lastErr)
		case <-time.After(delay):
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * cfg.BackoffFactor)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return fmt.Errorf("%s: max retry attempts exceeded: %w", serviceName, lastErr)
}

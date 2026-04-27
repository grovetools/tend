package retry

import (
	"context"
	"fmt"
	"time"
)

// Options configures retry behavior
type Options struct {
	MaxAttempts int
	Delay       time.Duration
	Multiplier  float64 // For exponential backoff
	MaxDelay    time.Duration
}

// DefaultOptions returns default retry options
func DefaultOptions() Options {
	return Options{
		MaxAttempts: 3,
		Delay:       1 * time.Second,
		Multiplier:  2.0,
		MaxDelay:    30 * time.Second,
	}
}

// Do retries a function until it succeeds
func Do(fn func() error, opts Options) error {
	return DoContext(context.Background(), fn, opts)
}

// DoContext retries with context support
func DoContext(ctx context.Context, fn func() error, opts Options) error {
	var lastErr error
	delay := opts.Delay

	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err

			if attempt < opts.MaxAttempts {
				// Calculate next delay
				if opts.Multiplier > 1 {
					delay = time.Duration(float64(delay) * opts.Multiplier)
					if delay > opts.MaxDelay {
						delay = opts.MaxDelay
					}
				}

				select {
				case <-ctx.Done():
					return fmt.Errorf("retry cancelled during delay: %w", ctx.Err())
				case <-time.After(delay):
				}
			}
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", opts.MaxAttempts, lastErr)
}

// WithTimeout retries until timeout
func WithTimeout(fn func() error, timeout time.Duration, interval time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Try immediately
	if err := fn(); err == nil {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout after %v", timeout)

		case <-ticker.C:
			if err := fn(); err == nil {
				return nil
			}
		}
	}
}

// UntilSuccess retries until the function succeeds or context is cancelled
func UntilSuccess(ctx context.Context, fn func() error, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			if err := fn(); err == nil {
				return nil
			}
		}
	}
}

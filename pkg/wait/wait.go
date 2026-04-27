package wait

import (
	"context"
	"fmt"
	"time"
)

// Condition represents a testable condition
type Condition func() (bool, error)

// ConditionWithMessage represents a condition with descriptive message
type ConditionWithMessage func() (bool, string, error)

// Options configures wait behavior
type Options struct {
	Timeout      time.Duration
	PollInterval time.Duration
	Immediate    bool // Check condition immediately before first sleep
}

// DefaultOptions returns default wait options
func DefaultOptions() Options {
	return Options{
		Timeout:      30 * time.Second,
		PollInterval: 500 * time.Millisecond,
		Immediate:    true,
	}
}

// For waits for a condition to become true
func For(condition Condition, opts Options) error {
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	return ForContext(ctx, condition, opts)
}

// ForContext waits for a condition with context
func ForContext(ctx context.Context, condition Condition, opts Options) error {
	if opts.Immediate {
		if ok, err := condition(); err != nil {
			return fmt.Errorf("condition check failed: %w", err)
		} else if ok {
			return nil
		}
	}

	ticker := time.NewTicker(opts.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for condition: %w", ctx.Err())

		case <-ticker.C:
			ok, err := condition()
			if err != nil {
				return fmt.Errorf("condition check failed: %w", err)
			}
			if ok {
				return nil
			}
		}
	}
}

// ForWithMessage waits for a condition and provides status messages
func ForWithMessage(condition ConditionWithMessage, opts Options) error {
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	return ForContextWithMessage(ctx, condition, opts)
}

// ForContextWithMessage waits with context and message updates
func ForContextWithMessage(ctx context.Context, condition ConditionWithMessage, opts Options) error {
	var lastMessage string

	checkCondition := func() error {
		ok, msg, err := condition()
		lastMessage = msg

		if err != nil {
			return fmt.Errorf("condition check failed: %w", err)
		}
		if ok {
			return nil
		}
		return fmt.Errorf("condition not met")
	}

	if opts.Immediate {
		if err := checkCondition(); err == nil {
			return nil
		}
	}

	ticker := time.NewTicker(opts.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for condition: %s (last status: %s)",
				ctx.Err(), lastMessage)

		case <-ticker.C:
			if err := checkCondition(); err == nil {
				return nil
			}
		}
	}
}

// Until waits until a condition becomes false
func Until(condition Condition, opts Options) error {
	invertedCondition := func() (bool, error) {
		ok, err := condition()
		return !ok, err
	}
	return For(invertedCondition, opts)
}

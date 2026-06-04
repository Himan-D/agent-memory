package resilience

import (
	"context"
	"math"
	"time"
)

type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    5 * time.Second,
	}
}

func Retry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	var err error
	for i := 0; i < cfg.MaxAttempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		delay := time.Duration(math.Pow(2, float64(i))) * cfg.BaseDelay
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return err
}

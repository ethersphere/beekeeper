package k8s

import (
	"io"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/logging"
	"k8s.io/client-go/util/flowcontrol"
)

// These option setters mutate unexported Client fields, so they are tested from
// within the package (the rest of pkg/k8s uses an external k8s_test package).

func TestWithLogger(t *testing.T) {
	t.Parallel()
	base := logging.New(io.Discard, 0)

	t.Run("sets_logger", func(t *testing.T) {
		newLogger := logging.New(io.Discard, 0)
		c := &Client{logger: base}
		WithLogger(newLogger)(c)
		if c.logger != newLogger {
			t.Error("logger not set to the provided logger")
		}
	})

	t.Run("nil_logger_ignored", func(t *testing.T) {
		c := &Client{logger: base}
		WithLogger(nil)(c)
		if c.logger != base {
			t.Error("nil logger should be ignored, logger changed")
		}
	})
}

func TestWithRequestLimiter(t *testing.T) {
	t.Parallel()
	baseLimiter := flowcontrol.NewTokenBucketRateLimiter(1, 1)

	t.Run("sets_limiter_and_max", func(t *testing.T) {
		limiter := flowcontrol.NewTokenBucketRateLimiter(50, 100)
		c := &Client{rateLimiter: baseLimiter, maxConcurrentRequests: 20}
		WithRequestLimiter(limiter, 5)(c)
		if c.rateLimiter != limiter {
			t.Error("rateLimiter not set to the provided limiter")
		}
		if c.maxConcurrentRequests != 5 {
			t.Errorf("maxConcurrentRequests expected: 5, got: %d", c.maxConcurrentRequests)
		}
	})

	t.Run("zero_max_allowed", func(t *testing.T) {
		limiter := flowcontrol.NewTokenBucketRateLimiter(50, 100)
		c := &Client{rateLimiter: baseLimiter, maxConcurrentRequests: 20}
		WithRequestLimiter(limiter, 0)(c)
		if c.maxConcurrentRequests != 0 {
			t.Errorf("maxConcurrentRequests expected: 0, got: %d", c.maxConcurrentRequests)
		}
	})

	t.Run("nil_limiter_ignored", func(t *testing.T) {
		c := &Client{rateLimiter: baseLimiter, maxConcurrentRequests: 20}
		WithRequestLimiter(nil, 7)(c)
		if c.rateLimiter != baseLimiter {
			t.Error("nil rateLimiter should be ignored, rateLimiter changed")
		}
		if c.maxConcurrentRequests != 7 {
			t.Errorf("maxConcurrentRequests expected: 7, got: %d", c.maxConcurrentRequests)
		}
	})

	t.Run("negative_max_ignored", func(t *testing.T) {
		limiter := flowcontrol.NewTokenBucketRateLimiter(50, 100)
		c := &Client{rateLimiter: baseLimiter, maxConcurrentRequests: 20}
		WithRequestLimiter(limiter, -1)(c)
		if c.maxConcurrentRequests != 20 {
			t.Errorf("negative maxConcurrentRequests should be ignored, got: %d", c.maxConcurrentRequests)
		}
		if c.rateLimiter != limiter {
			t.Error("rateLimiter should still be set")
		}
	})
}

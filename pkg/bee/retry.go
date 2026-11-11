package bee

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/swap"
)

const (
	defaultRetryDelay    = time.Second
	defaultMaxRetryDelay = 10 * time.Second
)

// retryRead calls fn up to c.retryCount times, doubling the wait between
// attempts from c.retryDelay up to c.maxRetryDelay. It stops early on errors
// that are not transient and when ctx is done. It must only wrap idempotent
// read calls: retrying a write may repeat its side effects.
func retryRead[T any](ctx context.Context, c *Client, op string, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	delay := c.retryDelay

	for attempt := 1; ; attempt++ {
		v, err := fn(ctx)
		if err == nil {
			return v, nil
		}
		if ctx.Err() != nil || attempt >= c.retryCount || !isRetryable(err) {
			return zero, err
		}

		c.log.Warningf("%s failed (attempt %d/%d), retrying in %s: %v", op, attempt, c.retryCount, delay, err)

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, fmt.Errorf("%w: last error: %w", ctx.Err(), err)
		case <-timer.C:
		}

		delay = min(delay*2, c.maxRetryDelay)
	}
}

// isRetryable reports whether err may be transient: network failures, 5xx
// and 429 responses are retried, while client errors and a missing swap
// client are not.
func isRetryable(err error) bool {
	var statusErr *api.HTTPStatusError
	if errors.As(err, &statusErr) {
		return statusErr.Code >= http.StatusInternalServerError || statusErr.Code == http.StatusTooManyRequests
	}

	var badRequestErr *swap.BadRequestError
	if errors.As(err, &badRequestErr) {
		return false
	}

	for _, permanent := range []error{
		swap.ErrNotSet,
		swap.ErrUnauthorized,
		swap.ErrForbidden,
		swap.ErrNotFound,
		swap.ErrMethodNotAllowed,
	} {
		if errors.Is(err, permanent) {
			return false
		}
	}

	return true
}

package builder

import (
	"context"
	"time"
)

// processQueueForSubmission checks for update signal and calls submit respecting provided context
func processQueueForSubmission(ctx context.Context, updateSignal chan struct{}, submit func()) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-updateSignal:
			submit()
		}
	}
}

// runRetryLoop calls retry periodically with the provided interval respecting context cancellation
func runRetryLoop(ctx context.Context, interval time.Duration, retry func()) bool {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return true
		case <-t.C:
			retry()
		}
	}
}

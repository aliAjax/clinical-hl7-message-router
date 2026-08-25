package infrastructure

import (
	"context"
	"time"
)

func Backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	return time.Duration(1<<min(attempt, 8)) * time.Second
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func WaitBackoff(_ context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	<-timer.C
	return nil
}

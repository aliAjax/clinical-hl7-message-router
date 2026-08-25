package infrastructure

import "time"

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

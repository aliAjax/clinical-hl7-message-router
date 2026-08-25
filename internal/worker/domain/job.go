package domain

import (
	"fmt"
	"time"
)

type Kind string

const (
	DeliveryRetry Kind = "delivery_retry"
	DeadReplay    Kind = "dead_replay"
	RetentionRun  Kind = "retention_run"
	TargetProbe   Kind = "target_probe"
)

type Job struct {
	ID          string
	Kind        Kind
	ReferenceID string
	Attempt     int
	MaxAttempts int
	RunAfter    time.Time
	CreatedAt   time.Time
	LastError   string
}

func (j Job) Validate() error {
	if j.ID == "" {
		return fmt.Errorf("job id is required")
	}
	switch j.Kind {
	case DeliveryRetry, DeadReplay, RetentionRun, TargetProbe:
	default:
		return fmt.Errorf("unsupported job kind %q", j.Kind)
	}
	if j.MaxAttempts < 1 {
		return fmt.Errorf("max attempts must be positive")
	}
	if j.Attempt < 0 || j.Attempt > j.MaxAttempts {
		return fmt.Errorf("attempt must be between zero and max attempts")
	}
	return nil
}

func (j Job) Ready(now time.Time) bool {
	return !j.RunAfter.After(now) && j.Attempt < j.MaxAttempts
}

func (j Job) Next(err error, now time.Time) Job {
	j.Attempt++
	if err != nil {
		j.LastError = err.Error()
	}
	delay := time.Second * time.Duration(1<<min(j.Attempt, 8))
	j.RunAfter = now.Add(delay)
	return j
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

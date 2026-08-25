package domain

import "time"

type Status string

const (
	Pending   Status = "pending"
	Delivered Status = "delivered"
	Failed    Status = "failed"
	Dead      Status = "dead"
)

type Delivery struct {
	ID, MessageID, TargetID string
	Status                  Status
	Attempts                int
	LastError               string
	UpdatedAt               time.Time
	Retryable               bool
}

func (d Delivery) NextAttempt(cause error, retryable bool, now time.Time) Delivery {
	d.Attempts++
	d.UpdatedAt = now
	d.Retryable = true
	if cause != nil {
		d.LastError = cause.Error()
		d.Status = Failed
	}
	return d
}

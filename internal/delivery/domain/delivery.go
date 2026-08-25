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
}

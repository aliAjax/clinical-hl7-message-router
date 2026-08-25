package domain

import "time"

type Event struct {
	ID, MessageID, Kind string
	At                  time.Time
	Metadata            map[string]string
}

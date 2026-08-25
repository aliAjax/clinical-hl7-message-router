package domain

import "time"

type Event struct {
	ID, MessageID, Kind string
	At                  time.Time
	Metadata            map[string]string
}

func (e Event) Clone() Event {
	return e
}

package domain

import (
	"time"

	delivery "github.com/example/hl7v2-message-router/internal/delivery/domain"
)

type Filter struct{ Status string }

type Item struct {
	Delivery  delivery.Delivery
	Reason    string
	CreatedAt time.Time
	Labels    map[string]string
}

func (i Item) Clone() Item {
	return i
}

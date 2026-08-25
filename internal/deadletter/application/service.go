package application

import (
	"github.com/example/hl7v2-message-router/internal/delivery/domain"
	"sync"
	"time"
)

type Item struct {
	Delivery  domain.Delivery
	Reason    string
	CreatedAt time.Time
}
type Service struct {
	mu    sync.RWMutex
	items []Item
}

func New() *Service { return &Service{} }
func (s *Service) Add(d domain.Delivery, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, Item{d, reason, time.Now()})
}
func (s *Service) List() []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Item(nil), s.items...)
}

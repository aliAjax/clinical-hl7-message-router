package application

import (
	deadletter "github.com/example/hl7v2-message-router/internal/deadletter/domain"
	delivery "github.com/example/hl7v2-message-router/internal/delivery/domain"
	"sync"
	"time"
)

type Service struct {
	mu    sync.RWMutex
	items []deadletter.Item
}

func New() *Service { return &Service{} }
func (s *Service) Add(d delivery.Delivery, reason string) {
	s.AddWithLabels(d, reason, nil)
}
func (s *Service) AddWithLabels(d delivery.Delivery, reason string, labels map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, deadletter.Item{Delivery: d, Reason: reason, CreatedAt: time.Now(), Labels: labels})
}
func (s *Service) List() []deadletter.Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]deadletter.Item(nil), s.items...)
}

package application

import (
	"github.com/example/hl7v2-message-router/internal/trace/domain"
	"sync"
	"time"
)

type Service struct {
	mu     sync.RWMutex
	events []domain.Event
}

func New() *Service { return &Service{} }
func (s *Service) Add(e domain.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.At.IsZero() {
		e.At = time.Now()
	}
	s.events = append(s.events, e)
}
func (s *Service) List(id string) []domain.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var o []domain.Event
	for _, e := range s.events {
		if id == "" || e.MessageID == id {
			o = append(o, e)
		}
	}
	return o
}

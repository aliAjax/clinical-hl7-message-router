package application

import (
	"context"
	"fmt"
	"github.com/example/hl7v2-message-router/internal/hl7/domain"
	"github.com/example/hl7v2-message-router/internal/routing/adapter"
	routing "github.com/example/hl7v2-message-router/internal/routing/domain"
	"github.com/example/hl7v2-message-router/internal/routing/infrastructure"
	"sync"
)

type Store struct {
	mu         sync.RWMutex
	Routes     map[string]routing.Rule
	Targets    map[string]routing.Target
	repository *infrastructure.Repository
	view       *adapter.RouteView
}

func NewStore() *Store {
	return &Store{Routes: map[string]routing.Rule{}, Targets: map[string]routing.Target{}, repository: infrastructure.NewRepository(), view: adapter.NewRouteView()}
}
func (s *Store) AddRoute(r routing.Rule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.Version == 0 {
		r.Version = 1
	}
	if r.State == "" {
		r.State = routing.RuleDraft
	}
	s.Routes[r.ID] = r
	_ = s.repository.Commit(r)
}
func (s *Store) AddTarget(t routing.Target) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !t.Healthy {
		t.Healthy = true
	}
	s.Targets[t.ID] = t
}
func (s *Store) Publish(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.Routes[id]
	if !ok {
		return fmt.Errorf("route %s not found", id)
	}
	r, err := r.Transition(routing.RulePublished)
	if err != nil {
		return err
	}
	s.Routes[id] = r
	s.view.Apply(r)
	return s.repository.Commit(r)
}
func (s *Store) Match(_ context.Context, m domain.Message) []routing.Target {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []routing.Target
	for _, r := range s.view.Published() {
		if !r.Published || !r.Matches(m.MessageType, m.Trigger, m.SendingFacility) {
			continue
		}
		for _, id := range r.TargetIDs {
			if t, ok := s.Targets[id]; ok {
				out = append(out, t)
			}
		}
	}
	return out
}

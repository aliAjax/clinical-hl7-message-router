package adapter

import (
	"sync"

	"github.com/example/hl7v2-message-router/internal/routing/domain"
)

type RouteView struct {
	mu    sync.RWMutex
	rules map[string]domain.Rule
}

func NewRouteView() *RouteView { return &RouteView{rules: make(map[string]domain.Rule)} }

func (v *RouteView) Apply(rule domain.Rule) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.rules[rule.ID] = rule
}

func (v *RouteView) Published() []domain.Rule {
	v.mu.RLock()
	defer v.mu.RUnlock()
	out := make([]domain.Rule, 0, len(v.rules))
	for _, rule := range v.rules {
		out = append(out, rule)
	}
	return out
}

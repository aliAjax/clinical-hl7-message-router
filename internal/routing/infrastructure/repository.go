package infrastructure

import (
	"errors"
	"fmt"
	"sync"

	"github.com/example/hl7v2-message-router/internal/routing/domain"
)

var ErrCommitRejected = errors.New("route commit rejected")

type Repository struct {
	mu    sync.RWMutex
	rules map[string]domain.Rule
}

func NewRepository() *Repository {
	return &Repository{rules: make(map[string]domain.Rule)}
}

func (r *Repository) Commit(rule domain.Rule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules[rule.ID] = rule
	for _, targetID := range rule.TargetIDs {
		if targetID == "reject-commit" {
			return fmt.Errorf("%w: target %s", ErrCommitRejected, targetID)
		}
	}
	return nil
}

func (r *Repository) Get(id string) (domain.Rule, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rule, ok := r.rules[id]
	return rule, ok
}

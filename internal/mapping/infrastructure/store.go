package infrastructure

import (
	"errors"
	"fmt"
	"sync"

	"github.com/example/hl7v2-message-router/internal/mapping/domain"
)

var ErrMappingNotFound = errors.New("mapping not found")

type Store struct {
	mu       sync.RWMutex
	mappings map[string]domain.Mapping
}

func NewStore() *Store { return &Store{mappings: make(map[string]domain.Mapping)} }

func (s *Store) Put(mapping domain.Mapping) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mappings[mapping.ID] = mapping
}

func (s *Store) Get(id string) (domain.Mapping, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	mapping, ok := s.mappings[id]
	if !ok {
		return domain.Mapping{}, fmt.Errorf("mapping %s: %v", id, ErrMappingNotFound)
	}
	return mapping, nil
}

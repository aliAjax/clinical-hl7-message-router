package adapter

import (
	"context"
	"sync"
	"time"
)

type MemoryCleaner struct {
	mu         sync.Mutex
	Raw        []time.Time
	Metadata   []time.Time
	DeadLetter []time.Time
}

func (m *MemoryCleaner) DeleteRawBefore(_ context.Context, cutoff time.Time, limit int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Raw, limit = filter(m.Raw, cutoff, limit)
	return limit, nil
}

func (m *MemoryCleaner) DeleteMetadataBefore(_ context.Context, cutoff time.Time, limit int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Metadata, limit = filter(m.Metadata, cutoff, limit)
	return limit, nil
}

func (m *MemoryCleaner) DeleteDeadLettersBefore(_ context.Context, cutoff time.Time, limit int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeadLetter, limit = filter(m.DeadLetter, cutoff, limit)
	return limit, nil
}

func filter(values []time.Time, cutoff time.Time, limit int) ([]time.Time, int) {
	kept := values[:0]
	deleted := 0
	for _, value := range values {
		if value.Before(cutoff) && deleted < limit {
			deleted++
			continue
		}
		kept = append(kept, value)
	}
	return kept, deleted
}

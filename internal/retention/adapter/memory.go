package adapter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/example/hl7v2-message-router/internal/retention/application"
)

type MemoryCleaner struct {
	mu         sync.Mutex
	Raw        []time.Time
	Metadata   []time.Time
	DeadLetter []time.Time
	active     int
}

func (m *MemoryCleaner) BeginBatch(_ context.Context, kind application.CleanupKind, cutoff time.Time, limit int) (application.Batch, error) {
	if limit < 1 {
		return nil, fmt.Errorf("batch limit must be positive")
	}
	m.mu.Lock()
	m.active++
	m.mu.Unlock()
	return &memoryBatch{cleaner: m, kind: kind, cutoff: cutoff, limit: limit}, nil
}

type memoryBatch struct {
	cleaner *MemoryCleaner
	kind    application.CleanupKind
	cutoff  time.Time
	limit   int
	closed  bool
}

func (b *memoryBatch) Delete(_ context.Context) (int, error) {
	m := b.cleaner
	m.mu.Lock()
	defer m.mu.Unlock()

	var values *[]time.Time
	switch b.kind {
	case application.CleanupRaw:
		values = &m.Raw
	case application.CleanupMetadata:
		values = &m.Metadata
	case application.CleanupDeadLetters:
		values = &m.DeadLetter
	default:
		return 0, fmt.Errorf("unknown cleanup kind %q", b.kind)
	}
	var deleted int
	*values, deleted = filter(*values, b.cutoff, b.limit)
	return deleted, nil
}

func (b *memoryBatch) Close() error {
	m := b.cleaner
	m.mu.Lock()
	defer m.mu.Unlock()
	if b.closed {
		return nil
	}
	m.active--
	b.closed = true
	return nil
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

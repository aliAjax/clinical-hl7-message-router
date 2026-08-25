package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/example/hl7v2-message-router/internal/delivery/domain"
	hl7 "github.com/example/hl7v2-message-router/internal/hl7/domain"
)

var ErrMessageNotFound = errors.New("message not found")

type MessageState string

const (
	MessageAccepted  MessageState = "accepted"
	MessageDelivered MessageState = "delivered"
	MessageFailed    MessageState = "failed"
)

type MessageRecord struct {
	ID              string            `json:"id"`
	IdempotencyKey  string            `json:"idempotency_key"`
	MessageType     string            `json:"message_type"`
	Trigger         string            `json:"trigger"`
	SendingFacility string            `json:"sending_facility"`
	RawDigest       string            `json:"raw_digest"`
	State           MessageState      `json:"state"`
	ReceivedAt      time.Time         `json:"received_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	DeliveryCount   int               `json:"delivery_count"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

type MessageStore interface {
	Create(context.Context, MessageRecord, hl7.Message) (MessageRecord, bool, error)
	Get(context.Context, string) (MessageRecord, hl7.Message, error)
	UpdateDeliveries(context.Context, string, []domain.Delivery) error
	List(context.Context, int) ([]MessageRecord, error)
}

type messageEntry struct {
	record  MessageRecord
	message hl7.Message
}

type MemoryMessageStore struct {
	mu      sync.RWMutex
	entries map[string]messageEntry
	byKey   map[string]string
	order   []string
	limit   int
}

func NewMemoryMessageStore(limit int) *MemoryMessageStore {
	if limit < 1 {
		limit = 10000
	}
	return &MemoryMessageStore{entries: make(map[string]messageEntry), byKey: make(map[string]string), limit: limit}
}

func (s *MemoryMessageStore) Create(_ context.Context, record MessageRecord, message hl7.Message) (MessageRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, exists := s.byKey[record.IdempotencyKey]; exists {
		return s.entries[id].record, false, nil
	}
	if record.ID == "" || record.IdempotencyKey == "" {
		return MessageRecord{}, false, fmt.Errorf("message id and idempotency key are required")
	}
	if _, exists := s.entries[record.ID]; exists {
		return MessageRecord{}, false, fmt.Errorf("message id %s already exists", record.ID)
	}
	now := time.Now().UTC()
	if record.ReceivedAt.IsZero() {
		record.ReceivedAt = now
	}
	record.UpdatedAt = now
	if record.State == "" {
		record.State = MessageAccepted
	}
	if record.Metadata == nil {
		record.Metadata = make(map[string]string)
	}
	s.entries[record.ID] = messageEntry{record: record, message: message}
	s.byKey[record.IdempotencyKey] = record.ID
	s.order = append(s.order, record.ID)
	for len(s.order) > s.limit {
		oldest := s.order[0]
		s.order = s.order[1:]
		entry := s.entries[oldest]
		delete(s.byKey, entry.record.IdempotencyKey)
		delete(s.entries, oldest)
	}
	return record, true, nil
}

func (s *MemoryMessageStore) Get(_ context.Context, id string) (MessageRecord, hl7.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.entries[id]
	if !ok {
		return MessageRecord{}, hl7.Message{}, fmt.Errorf("%w: %s", ErrMessageNotFound, id)
	}
	return entry.record, entry.message, nil
}

func (s *MemoryMessageStore) UpdateDeliveries(_ context.Context, id string, deliveries []domain.Delivery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrMessageNotFound, id)
	}
	entry.record.DeliveryCount += len(deliveries)
	entry.record.UpdatedAt = time.Now().UTC()
	entry.record.State = MessageDelivered
	for _, delivery := range deliveries {
		if delivery.Status == domain.Failed || delivery.Status == domain.Dead {
			entry.record.State = MessageFailed
			break
		}
	}
	s.entries[id] = entry
	return nil
}

func (s *MemoryMessageStore) List(_ context.Context, limit int) ([]MessageRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit < 1 || limit > len(s.order) {
		limit = len(s.order)
	}
	out := make([]MessageRecord, 0, limit)
	for i := len(s.order) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, s.entries[s.order[i]].record)
	}
	return out, nil
}

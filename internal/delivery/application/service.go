package application

import (
	"context"
	"fmt"
	"github.com/example/hl7v2-message-router/internal/delivery/domain"
	"github.com/example/hl7v2-message-router/internal/delivery/infrastructure"
	hl7 "github.com/example/hl7v2-message-router/internal/hl7/domain"
	routing "github.com/example/hl7v2-message-router/internal/routing/domain"
	"net"
	"strings"
	"sync"
	"time"
)

type Connector interface {
	Send(context.Context, routing.Target, hl7.Message) (string, error)
}
type MemoryConnector struct{}

func (MemoryConnector) Send(ctx context.Context, t routing.Target, m hl7.Message) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	if strings.HasPrefix(t.Address, "fail://") {
		return "", fmt.Errorf("target unavailable")
	}
	return "AA", nil
}

type Service struct {
	mu        sync.RWMutex
	items     map[string]domain.Delivery
	connector Connector
	policy    RetryPolicy
}

type RetryPolicy struct {
	MaximumAttempts int
	Backoff         func(int) time.Duration
}

func New(c Connector) *Service { return NewWithPolicy(c, RetryPolicy{MaximumAttempts: 3}) }
func NewWithPolicy(c Connector, policy RetryPolicy) *Service {
	if policy.MaximumAttempts < 1 {
		policy.MaximumAttempts = 1
	}
	if policy.Backoff == nil {
		policy.Backoff = infrastructure.Backoff
	}
	return &Service{items: map[string]domain.Delivery{}, connector: c, policy: policy}
}
func (s *Service) Deliver(ctx context.Context, m hl7.Message, targets []routing.Target) []domain.Delivery {
	var out []domain.Delivery
	for _, t := range targets {
		d := domain.Delivery{ID: m.IdempotencyKey() + "-" + t.ID, MessageID: m.IdempotencyKey(), TargetID: t.ID, Status: domain.Pending, UpdatedAt: time.Now()}
		for d.Attempts < s.policy.MaximumAttempts {
			ack, e := s.connector.Send(context.Background(), t, m)
			d = d.NextAttempt(e, true, time.Now())
			if e == nil && ack == "AA" {
				d.Status = domain.Delivered
				d.LastError = ""
				d.Retryable = false
				break
			}
			d.Status = domain.Failed
			if e != nil {
				d.LastError = e.Error()
			} else {
				d.LastError = fmt.Sprintf("negative ACK %q", ack)
			}
			if ctx.Err() != nil {
				d.LastError = ctx.Err().Error()
				d.Retryable = false
				break
			}
			if !d.Retryable {
				break
			}
			if d.Attempts < s.policy.MaximumAttempts {
				_ = infrastructure.WaitBackoff(context.Background(), s.policy.Backoff(d.Attempts))
			}
		}
		if d.Status != domain.Delivered && d.Attempts >= s.policy.MaximumAttempts {
			d.Status = domain.Dead
		}
		s.mu.Lock()
		s.items[d.ID] = d
		s.mu.Unlock()
		out = append(out, d)
	}
	return out
}
func (s *Service) List() []domain.Delivery {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o := make([]domain.Delivery, 0, len(s.items))
	for _, d := range s.items {
		o = append(o, d)
	}
	return o
}
func Probe(ctx context.Context, address string) error {
	d := net.Dialer{Timeout: time.Second}
	c, e := d.DialContext(ctx, "tcp", address)
	if e == nil {
		c.Close()
	}
	return e
}

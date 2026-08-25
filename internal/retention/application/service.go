package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/example/hl7v2-message-router/internal/retention/domain"
)

type Cleaner interface {
	DeleteRawBefore(context.Context, time.Time, int) (int, error)
	DeleteMetadataBefore(context.Context, time.Time, int) (int, error)
	DeleteDeadLettersBefore(context.Context, time.Time, int) (int, error)
}

type Result struct {
	RawDeleted         int
	MetadataDeleted    int
	DeadLettersDeleted int
	StartedAt          time.Time
	CompletedAt        time.Time
}

type Service struct {
	policy  domain.Policy
	cleaner Cleaner
	clock   func() time.Time
	logger  *slog.Logger
}

func New(policy domain.Policy, cleaner Cleaner, logger *slog.Logger) (*Service, error) {
	if err := policy.Validate(); err != nil {
		return nil, fmt.Errorf("retention policy: %w", err)
	}
	if cleaner == nil {
		return nil, fmt.Errorf("cleaner is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{policy: policy, cleaner: cleaner, clock: time.Now, logger: logger}, nil
}

func (s *Service) Run(ctx context.Context) (Result, error) {
	now := s.clock().UTC()
	result := Result{StartedAt: now}
	var err error
	result.RawDeleted, err = s.cleaner.DeleteRawBefore(ctx, now.Add(-s.policy.RawMessageTTL), s.policy.BatchSize)
	if err != nil {
		return result, fmt.Errorf("delete expired raw messages: %w", err)
	}
	result.MetadataDeleted, err = s.cleaner.DeleteMetadataBefore(ctx, now.Add(-s.policy.MetadataTTL), s.policy.BatchSize)
	if err != nil {
		return result, fmt.Errorf("delete expired metadata: %w", err)
	}
	result.DeadLettersDeleted, err = s.cleaner.DeleteDeadLettersBefore(ctx, now.Add(-s.policy.DeadLetterTTL), s.policy.BatchSize)
	if err != nil {
		return result, fmt.Errorf("delete expired dead letters: %w", err)
	}
	result.CompletedAt = s.clock().UTC()
	s.logger.InfoContext(ctx, "retention run completed", "raw_deleted", result.RawDeleted, "metadata_deleted", result.MetadataDeleted, "dead_letters_deleted", result.DeadLettersDeleted)
	return result, nil
}

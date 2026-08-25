package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/example/hl7v2-message-router/internal/retention/domain"
)

type CleanupKind string

const (
	CleanupRaw         CleanupKind = "raw"
	CleanupMetadata    CleanupKind = "metadata"
	CleanupDeadLetters CleanupKind = "dead_letters"
)

type Batch interface {
	Delete(context.Context) (int, error)
	Close() error
}

type Cleaner interface {
	BeginBatch(context.Context, CleanupKind, time.Time, int) (Batch, error)
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

type cleanupStage struct {
	kind   CleanupKind
	cutoff time.Time
	store  func(*Result, int)
}

func (s *Service) Run(ctx context.Context) (result Result, err error) {
	now := s.clock().UTC()
	result.StartedAt = now
	stages := []cleanupStage{
		{kind: CleanupRaw, cutoff: now.Add(-s.policy.RawMessageTTL), store: func(r *Result, n int) { r.RawDeleted = n }},
		{kind: CleanupMetadata, cutoff: now.Add(-s.policy.MetadataTTL), store: func(r *Result, n int) { r.MetadataDeleted = n }},
		{kind: CleanupDeadLetters, cutoff: now.Add(-s.policy.DeadLetterTTL), store: func(r *Result, n int) { r.DeadLettersDeleted = n }},
	}

	for _, stage := range stages {
		stageErr := s.runStage(ctx, stage, &result)
		if stageErr != nil {
			err = errors.Join(err, stageErr)
			return result, err
		}
	}

	result.CompletedAt = s.clock().UTC()
	s.logger.InfoContext(ctx, "retention run completed", "raw_deleted", result.RawDeleted, "metadata_deleted", result.MetadataDeleted, "dead_letters_deleted", result.DeadLettersDeleted)
	return result, err
}

// runStage opens a single cleanup batch, deletes the expired rows for the
// stage, and guarantees the batch is released before returning. The deferred
// Close runs at the end of the stage (not the end of the whole run) so batch
// resources do not accumulate across stages. Any delete error halts the run
// instead of continuing into later stages while holding a half-open batch.
// Close errors are joined with (not assigned over) a prior delete error so the
// delete failure is preserved in the returned error.
func (s *Service) runStage(ctx context.Context, stage cleanupStage, result *Result) (err error) {
	batch, beginErr := s.cleaner.BeginBatch(ctx, stage.kind, stage.cutoff, s.policy.BatchSize)
	if beginErr != nil {
		return fmt.Errorf("begin %s cleanup: %w", stage.kind, beginErr)
	}
	defer func() {
		if closeErr := batch.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close %s cleanup: %w", stage.kind, closeErr))
		}
	}()

	deleted, deleteErr := batch.Delete(ctx)
	if deleteErr != nil {
		return fmt.Errorf("delete %s: %w", stage.kind, deleteErr)
	}

	s.logger.DebugContext(ctx, "retention batch deleted", "kind", stage.kind, "deleted", deleted)
	stage.store(result, deleted)
	return nil
}

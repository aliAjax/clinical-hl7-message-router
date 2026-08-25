package application

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/example/hl7v2-message-router/internal/retention/domain"
)

var (
	errBeginResource = errors.New("batch resource unavailable")
	errDeleteBatch   = errors.New("delete batch failed")
	errCloseBatch    = errors.New("close batch failed")
)

type scriptedCleaner struct {
	batches       map[CleanupKind]*scriptedBatch
	beginErrors   map[CleanupKind]error
	begun         []CleanupKind
	open          int
	rejectOverlap bool
}

func (c *scriptedCleaner) BeginBatch(_ context.Context, kind CleanupKind, _ time.Time, _ int) (Batch, error) {
	c.begun = append(c.begun, kind)
	if err := c.beginErrors[kind]; err != nil {
		return nil, err
	}
	if c.rejectOverlap && c.open != 0 {
		return nil, errBeginResource
	}
	b := c.batches[kind]
	if b == nil {
		b = &scriptedBatch{deleted: 1}
		c.batches[kind] = b
	}
	b.owner = c
	c.open++
	return b, nil
}

type scriptedBatch struct {
	owner     *scriptedCleaner
	deleted   int
	deleteErr error
	closeErr  error
	closed    bool
}

func (b *scriptedBatch) Delete(context.Context) (int, error) {
	return b.deleted, b.deleteErr
}

func (b *scriptedBatch) Close() error {
	if !b.closed {
		b.closed = true
		b.owner.open--
	}
	return b.closeErr
}

func newScriptedCleaner() *scriptedCleaner {
	return &scriptedCleaner{
		batches: map[CleanupKind]*scriptedBatch{
			CleanupRaw:         {deleted: 1},
			CleanupMetadata:    {deleted: 2},
			CleanupDeadLetters: {deleted: 3},
		},
		beginErrors: make(map[CleanupKind]error),
	}
}

func newRetentionService(t *testing.T, cleaner Cleaner) *Service {
	t.Helper()
	service, err := New(domain.DefaultPolicy(), cleaner, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	service.clock = func() time.Time { return time.Unix(1_700_000_000, 0) }
	return service
}

func TestCleanupReleasesEachBatch(t *testing.T) {
	cleaner := newScriptedCleaner()
	cleaner.rejectOverlap = true
	result, err := newRetentionService(t, cleaner).Run(context.Background())
	if err != nil {
		t.Fatalf("cleanup returned error: %v", err)
	}
	if cleaner.open != 0 || result.RawDeleted != 1 || result.MetadataDeleted != 2 || result.DeadLettersDeleted != 3 {
		t.Fatalf("open=%d result=%+v", cleaner.open, result)
	}
}

func TestCleanupPreservesPrimaryError(t *testing.T) {
	cleaner := newScriptedCleaner()
	cleaner.batches[CleanupRaw].deleteErr = errDeleteBatch
	cleaner.batches[CleanupRaw].closeErr = errCloseBatch
	_, err := newRetentionService(t, cleaner).Run(context.Background())
	if !errors.Is(err, errDeleteBatch) || !errors.Is(err, errCloseBatch) {
		t.Fatalf("error chain lost delete or close failure: %v", err)
	}
}

func TestCleanupStopsAfterFailure(t *testing.T) {
	cleaner := newScriptedCleaner()
	cleaner.batches[CleanupMetadata].deleteErr = errDeleteBatch
	_, err := newRetentionService(t, cleaner).Run(context.Background())
	if !errors.Is(err, errDeleteBatch) {
		t.Fatalf("delete error = %v", err)
	}
	if len(cleaner.begun) != 2 {
		t.Fatalf("began %d stages after delete failure: %v", len(cleaner.begun), cleaner.begun)
	}
}

func TestCleanupStopsAfterBeginFailure(t *testing.T) {
	cleaner := newScriptedCleaner()
	cleaner.beginErrors[CleanupMetadata] = errBeginResource
	_, err := newRetentionService(t, cleaner).Run(context.Background())
	if !errors.Is(err, errBeginResource) {
		t.Fatalf("begin error = %v", err)
	}
	if len(cleaner.begun) != 2 {
		t.Fatalf("began %d stages after begin failure: %v", len(cleaner.begun), cleaner.begun)
	}
}

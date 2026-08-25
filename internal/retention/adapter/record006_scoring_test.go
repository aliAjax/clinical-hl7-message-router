package adapter

import (
	"context"
	"testing"
	"time"

	"github.com/example/hl7v2-message-router/internal/retention/application"
)

func TestBatchCloseReleasesExactlyOnce(t *testing.T) {
	cleaner := &MemoryCleaner{}
	batch, err := cleaner.BeginBatch(context.Background(), application.CleanupRaw, time.Now(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := batch.Close(); err != nil {
		t.Fatal(err)
	}
	if err := batch.Close(); err != nil {
		t.Fatal(err)
	}
	if cleaner.active != 0 {
		t.Fatalf("active batches after repeated close = %d", cleaner.active)
	}
}

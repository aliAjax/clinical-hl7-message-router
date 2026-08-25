package main

import (
	"context"
	"errors"
	"testing"

	hl7 "github.com/example/hl7v2-message-router/internal/hl7/application"
)

func TestHandlerHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (handler{p: hl7.NewParser()}).Handle(ctx, "not an HL7 message")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("handler error = %v", err)
	}
}

package application

import (
	"context"
	hl7 "github.com/example/hl7v2-message-router/internal/hl7/domain"
	routing "github.com/example/hl7v2-message-router/internal/routing/domain"
	"testing"
)

func TestFailureRetriesThenDeadLetters(t *testing.T) {
	service := NewWithPolicy(MemoryConnector{}, RetryPolicy{MaximumAttempts: 3})
	message := hl7.Message{ControlID: "FAIL-001", Raw: "message"}
	deliveries := service.Deliver(context.Background(), message, []routing.Target{{ID: "down", Address: "fail://down"}})
	if len(deliveries) != 1 {
		t.Fatalf("deliveries = %d", len(deliveries))
	}
	if deliveries[0].Status != "dead" || deliveries[0].Attempts != 3 {
		t.Fatalf("delivery = %#v", deliveries[0])
	}
}

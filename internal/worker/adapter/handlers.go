package adapter

import (
	"context"
	"fmt"

	"github.com/example/hl7v2-message-router/internal/worker/domain"
)

type Dispatcher struct {
	Delivery func(context.Context, string) error
	Replay   func(context.Context, string) error
	Probe    func(context.Context, string) error
	Cleanup  func(context.Context) error
}

func (d Dispatcher) Handle(ctx context.Context, job domain.Job) error {
	switch job.Kind {
	case domain.DeliveryRetry:
		if d.Delivery == nil {
			return fmt.Errorf("delivery retry handler is not configured")
		}
		return d.Delivery(ctx, job.ReferenceID)
	case domain.DeadReplay:
		if d.Replay == nil {
			return fmt.Errorf("dead replay handler is not configured")
		}
		return d.Replay(ctx, job.ReferenceID)
	case domain.TargetProbe:
		if d.Probe == nil {
			return fmt.Errorf("target probe handler is not configured")
		}
		return d.Probe(ctx, job.ReferenceID)
	case domain.RetentionRun:
		if d.Cleanup == nil {
			return fmt.Errorf("retention handler is not configured")
		}
		return d.Cleanup(ctx)
	default:
		return fmt.Errorf("unsupported job kind %q", job.Kind)
	}
}

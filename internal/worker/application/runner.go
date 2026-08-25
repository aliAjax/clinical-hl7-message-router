package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/example/hl7v2-message-router/internal/worker/domain"
)

var ErrQueueClosed = errors.New("worker queue closed")

type Handler interface {
	Handle(context.Context, domain.Job) error
}

type HandlerFunc func(context.Context, domain.Job) error

func (f HandlerFunc) Handle(ctx context.Context, job domain.Job) error { return f(ctx, job) }

type Queue interface {
	Enqueue(context.Context, domain.Job) error
	Dequeue(context.Context) (domain.Job, error)
	Complete(context.Context, domain.Job) error
	Fail(context.Context, domain.Job, error) error
}

type Runner struct {
	queue       Queue
	handlers    map[domain.Kind]Handler
	concurrency int
	logger      *slog.Logger
	wg          sync.WaitGroup
}

func NewRunner(queue Queue, concurrency int, logger *slog.Logger) (*Runner, error) {
	if queue == nil {
		return nil, fmt.Errorf("queue is required")
	}
	if concurrency < 1 || concurrency > 128 {
		return nil, fmt.Errorf("concurrency must be between 1 and 128")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Runner{queue: queue, handlers: make(map[domain.Kind]Handler), concurrency: concurrency, logger: logger}, nil
}

func (r *Runner) Register(kind domain.Kind, handler Handler) error {
	if handler == nil {
		return fmt.Errorf("handler for %s is nil", kind)
	}
	if _, exists := r.handlers[kind]; exists {
		return fmt.Errorf("handler for %s is already registered", kind)
	}
	r.handlers[kind] = handler
	return nil
}

func (r *Runner) Run(ctx context.Context) {
	for index := 0; index < r.concurrency; index++ {
		go r.worker(ctx, index)
	}
}

func (r *Runner) Wait() { r.wg.Wait() }

func (r *Runner) worker(ctx context.Context, index int) {
	for {
		job, err := r.queue.Dequeue(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, ErrQueueClosed) {
				return
			}
			r.logger.ErrorContext(ctx, "dequeue job", "worker", index, "error", err)
			continue
		}
		handler, ok := r.handlers[job.Kind]
		if !ok {
			err = fmt.Errorf("no handler registered for job kind %s", job.Kind)
		} else {
			err = handler.Handle(ctx, job)
		}
		if err != nil {
			if failErr := r.queue.Fail(ctx, job, err); failErr != nil {
				r.logger.ErrorContext(ctx, "record failed job", "job_id", job.ID, "error", failErr)
			}
			continue
		}
		if err := r.queue.Complete(ctx, job); err != nil {
			r.logger.ErrorContext(ctx, "complete job", "job_id", job.ID, "error", err)
		}
	}
}

type MemoryQueue struct {
	mu      sync.Mutex
	wakeup  chan struct{}
	items   []domain.Job
	closed  bool
	waiting int
	now     func() time.Time
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{wakeup: make(chan struct{}, 1), now: time.Now}
}

func (q *MemoryQueue) Enqueue(_ context.Context, job domain.Job) error {
	if err := job.Validate(); err != nil {
		return err
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrQueueClosed
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = q.now().UTC()
	}
	q.items = append(q.items, job)
	q.signal()
	return nil
}

func (q *MemoryQueue) Dequeue(ctx context.Context) (domain.Job, error) {
	for {
		q.mu.Lock()
		if q.closed {
			q.mu.Unlock()
			return domain.Job{}, ErrQueueClosed
		}
		now := q.now()
		for index, job := range q.items {
			if job.Ready(now) {
				q.items = append(q.items[:index], q.items[index+1:]...)
				q.mu.Unlock()
				return job, nil
			}
		}
		q.waiting++
		q.mu.Unlock()
		select {
		case <-ctx.Done():
			q.mu.Lock()
			q.waiting--
			q.mu.Unlock()
			return domain.Job{}, ctx.Err()
		case <-q.wakeup:
		case <-time.After(100 * time.Millisecond):
		}
		q.mu.Lock()
		q.waiting--
		q.mu.Unlock()
	}
}

func (q *MemoryQueue) Complete(_ context.Context, _ domain.Job) error { return nil }

func (q *MemoryQueue) Fail(ctx context.Context, job domain.Job, cause error) error {
	next := job.Next(cause, q.now())
	if next.Attempt >= next.MaxAttempts {
		return nil
	}
	return q.Enqueue(ctx, next)
}

func (q *MemoryQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.signal()
}

func (q *MemoryQueue) signal() {
	select {
	case q.wakeup <- struct{}{}:
	default:
	}
}

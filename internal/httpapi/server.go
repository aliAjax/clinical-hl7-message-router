package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	dead "github.com/example/hl7v2-message-router/internal/deadletter/application"
	delivery "github.com/example/hl7v2-message-router/internal/delivery/application"
	hl7 "github.com/example/hl7v2-message-router/internal/hl7/application"
	"github.com/example/hl7v2-message-router/internal/routing/application"
	routing "github.com/example/hl7v2-message-router/internal/routing/domain"
	trace "github.com/example/hl7v2-message-router/internal/trace/application"
	tracedomain "github.com/example/hl7v2-message-router/internal/trace/domain"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type Server struct {
	Parser           *hl7.Parser
	Routes           *application.Store
	Delivery         *delivery.Service
	Dead             *dead.Service
	Trace            *trace.Service
	Messages         trace.MessageStore
	Logger           *slog.Logger
	MiddlewareConfig MiddlewareConfig
	ready            atomic.Bool
	count            atomic.Uint64
}
type messageReq struct {
	Raw       string `json:"raw"`
	MappingID string `json:"mapping_id"`
}

func New(p *hl7.Parser, r *application.Store, d *delivery.Service, dl *dead.Service, t *trace.Service, l *slog.Logger) *Server {
	s := &Server{Parser: p, Routes: r, Delivery: d, Dead: dl, Trace: t, Messages: trace.NewMemoryMessageStore(10000), Logger: l}
	s.ready.Store(true)
	return s
}
func (s *Server) Handler() http.Handler {
	config := s.MiddlewareConfig
	if config == (MiddlewareConfig{}) {
		config = DefaultMiddlewareConfig()
	}
	middleware := NewMiddleware(config, s.Logger)
	return middleware.Wrap(http.HandlerFunc(s.route))
}
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := fmt.Sprintf("req-%d", time.Now().UnixNano())
		w.Header().Set("X-Request-ID", rid)
		if r.Method != "GET" && r.Header.Get("Content-Type") != "application/json" {
			w.Header().Set("Content-Type", "application/json")
		}
		defer func() {
			if recover() != nil {
				write(w, 500, map[string]string{"error": "internal error"})
			}
		}()
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	s.count.Add(1)
	p := strings.Trim(r.URL.Path, "/")
	switch {
	case r.URL.Path == "/healthz":
		write(w, 200, map[string]string{"status": "ok"})
	case r.URL.Path == "/readyz":
		if s.ready.Load() {
			write(w, 200, map[string]string{"status": "ready"})
		} else {
			write(w, 503, map[string]string{"status": "not_ready"})
		}
	case r.URL.Path == "/metrics":
		write(w, 200, fmt.Sprintf("hl7_messages_total %d\n", s.count.Load()))
	case r.Method == "POST" && p == "v1/targets":
		s.createTarget(w, r)
	case r.Method == "POST" && p == "v1/routes":
		s.createRoute(w, r)
	case r.Method == "POST" && strings.HasPrefix(p, "v1/routes/") && strings.HasSuffix(p, "/publish"):
		id := strings.Split(p, "/")[2]
		if e := s.Routes.Publish(id); e != nil {
			write(w, 404, map[string]string{"error": e.Error()})
		} else {
			write(w, 200, map[string]string{"status": "published"})
		}
	case r.Method == "POST" && strings.HasPrefix(p, "v1/routes/") && strings.HasSuffix(p, "/validate"):
		id := strings.Split(p, "/")[2]
		result, err := s.Routes.Validate(id)
		if err != nil {
			write(w, 404, map[string]string{"error": err.Error()})
		} else if !result.Valid {
			write(w, 422, result)
		} else {
			write(w, 200, result)
		}
	case r.Method == "POST" && p == "v1/messages":
		s.createMessage(w, r)
	case r.Method == "POST" && strings.HasPrefix(p, "v1/messages/") && strings.HasSuffix(p, "/replay"):
		s.replayMessage(w, r, strings.Split(p, "/")[2])
	case r.Method == "GET" && p == "v1/dead-letters":
		write(w, 200, s.Dead.List())
	case r.Method == "GET" && strings.HasPrefix(p, "v1/messages/"):
		id := strings.Split(p, "/")[2]
		record, _, err := s.Messages.Get(r.Context(), id)
		if err != nil {
			writeMessageError(w, r, err)
			return
		}
		write(w, 200, map[string]any{"message": record, "deliveries": s.Delivery.List(), "trace": s.Trace.List(id)})
	default:
		write(w, 404, map[string]string{"error": "not found"})
	}
}

func writeMessageError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, trace.ErrMessageNotFound) || errors.Is(err, context.DeadlineExceeded) {
		status = http.StatusInternalServerError
	}
	write(w, status, map[string]string{"error": err.Error()})
}
func (s *Server) createTarget(w http.ResponseWriter, r *http.Request) {
	var t routing.Target
	if json.NewDecoder(r.Body).Decode(&t) != nil || t.ID == "" {
		write(w, 400, map[string]string{"error": "id required"})
		return
	}
	s.Routes.AddTarget(t)
	write(w, 201, t)
}
func (s *Server) createRoute(w http.ResponseWriter, r *http.Request) {
	var v routing.Rule
	if json.NewDecoder(r.Body).Decode(&v) != nil || v.ID == "" {
		write(w, 400, map[string]string{"error": "id required"})
		return
	}
	s.Routes.AddRoute(v)
	write(w, 201, v)
}
func (s *Server) createMessage(w http.ResponseWriter, r *http.Request) {
	var q messageReq
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		write(w, 400, map[string]string{"error": "invalid json"})
		return
	}
	m, e := s.Parser.Parse(q.Raw)
	if e != nil {
		write(w, 422, map[string]string{"error": e.Error()})
		return
	}
	if es := hl7.Validate(m); len(es) > 0 {
		write(w, 422, map[string]string{"error": es[0].Error()})
		return
	}
	m.ID = m.IdempotencyKey()
	digest := sha256.Sum256([]byte(m.Raw))
	record := trace.MessageRecord{ID: m.ID, IdempotencyKey: m.IdempotencyKey(), MessageType: m.MessageType, Trigger: m.Trigger, SendingFacility: m.SendingFacility, RawDigest: fmt.Sprintf("sha256:%x", digest), ReceivedAt: m.ReceivedAt}
	stored, created, err := s.Messages.Create(r.Context(), record, m)
	if err != nil {
		write(w, 500, map[string]string{"error": "store message failed"})
		return
	}
	if !created {
		write(w, 200, map[string]any{"id": stored.ID, "duplicate": true, "state": stored.State})
		return
	}
	targets := s.Routes.Match(r.Context(), m)
	ds := s.Delivery.Deliver(r.Context(), m, targets)
	_ = s.Messages.UpdateDeliveries(r.Context(), m.ID, ds)
	s.Trace.Add(traceEvent(m, ds))
	for _, d := range ds {
		if d.Status == "dead" {
			s.Dead.Add(d, d.LastError)
		}
	}
	write(w, 202, map[string]any{"id": m.ID, "targets": len(targets), "deliveries": ds})
}

func (s *Server) replayMessage(w http.ResponseWriter, r *http.Request, id string) {
	record, message, err := s.Messages.Get(r.Context(), id)
	if err != nil {
		write(w, 404, map[string]string{"error": err.Error()})
		return
	}
	targets := s.Routes.Match(r.Context(), message)
	deliveries := s.Delivery.Deliver(r.Context(), message, targets)
	_ = s.Messages.UpdateDeliveries(r.Context(), id, deliveries)
	s.Trace.Add(tracedomain.Event{ID: fmt.Sprintf("replay-%d", time.Now().UnixNano()), MessageID: id, Kind: "replay", Metadata: map[string]string{"previous_state": string(record.State)}})
	write(w, 202, map[string]any{"id": id, "replayed": true, "deliveries": deliveries})
}
func traceEvent(m interface{ IdempotencyKey() string }, ds any) tracedomain.Event {
	return tracedomain.Event{ID: fmt.Sprintf("trace-%d", time.Now().UnixNano()), MessageID: m.IdempotencyKey(), Kind: "delivery", Metadata: map[string]string{"result": fmt.Sprintf("%v", ds)}}
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

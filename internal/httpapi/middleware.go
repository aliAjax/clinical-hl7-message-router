package httpapi

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type contextKey string

const requestIDKey contextKey = "request-id"

type MiddlewareConfig struct {
	AuthToken       string
	Timeout         time.Duration
	MaximumBodySize int64
	RatePerSecond   float64
	Burst           float64
	TrustProxy      bool
}

func DefaultMiddlewareConfig() MiddlewareConfig {
	return MiddlewareConfig{Timeout: 15 * time.Second, MaximumBodySize: 1024 * 1024, RatePerSecond: 50, Burst: 100}
}

type clientBucket struct {
	tokens float64
	last   time.Time
}

type Middleware struct {
	config  MiddlewareConfig
	logger  *slog.Logger
	mu      sync.Mutex
	clients map[string]clientBucket
	now     func() time.Time
}

func NewMiddleware(config MiddlewareConfig, logger *slog.Logger) *Middleware {
	if config.Timeout <= 0 {
		config.Timeout = 15 * time.Second
	}
	if config.MaximumBodySize <= 0 {
		config.MaximumBodySize = 1024 * 1024
	}
	if config.RatePerSecond <= 0 {
		config.RatePerSecond = 50
	}
	if config.Burst <= 0 {
		config.Burst = 100
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Middleware{config: config, logger: logger, clients: make(map[string]clientBucket), now: time.Now}
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return m.recover(m.requestID(m.accessLog(m.rateLimit(m.authenticate(m.limitBody(m.timeout(next)))))))
}

func (m *Middleware) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if id == "" || len(id) > 128 {
			id = fmt.Sprintf("req-%d", m.now().UnixNano())
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

func (m *Middleware) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.config.AuthToken == "" || publicEndpoint(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		provided := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if len(provided) != len(m.config.AuthToken) || subtle.ConstantTimeCompare([]byte(provided), []byte(m.config.AuthToken)) != 1 {
			write(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized", "request_id": RequestID(r.Context())})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func publicEndpoint(path string) bool {
	return path == "/healthz" || path == "/readyz" || path == "/metrics"
}

func (m *Middleware) limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, m.config.MaximumBodySize)
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) timeout(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), m.config.Timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				m.logger.ErrorContext(r.Context(), "http panic recovered", "request_id", RequestID(r.Context()), "panic", fmt.Sprint(recovered))
				write(w, http.StatusInternalServerError, map[string]string{"error": "internal error", "request_id": RequestID(r.Context())})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client := m.clientAddress(r)
		now := m.now()
		m.mu.Lock()
		bucket := m.clients[client]
		if bucket.last.IsZero() {
			bucket = clientBucket{tokens: m.config.Burst, last: now}
		}
		elapsed := now.Sub(bucket.last).Seconds()
		bucket.tokens = minFloat(m.config.Burst, bucket.tokens+elapsed*m.config.RatePerSecond)
		bucket.last = now
		allowed := bucket.tokens >= 1
		if allowed {
			bucket.tokens--
		}
		m.clients[client] = bucket
		m.mu.Unlock()
		if !allowed {
			w.Header().Set("Retry-After", "1")
			write(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded", "request_id": RequestID(r.Context())})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) clientAddress(r *http.Request) string {
	if m.config.TrustProxy {
		if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
			return forwarded
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(data)
	r.bytes += n
	return n, err
}

func (m *Middleware) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := m.now()
		recorder := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		m.logger.InfoContext(r.Context(), "http request", "request_id", RequestID(r.Context()), "method", r.Method, "path", r.URL.Path, "status", recorder.status, "bytes", recorder.bytes, "duration_ms", m.now().Sub(started).Milliseconds())
	})
}

func minFloat(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

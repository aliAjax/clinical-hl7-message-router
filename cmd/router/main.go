package main

import (
	"context"
	"github.com/example/hl7v2-message-router/internal/config"
	dead "github.com/example/hl7v2-message-router/internal/deadletter/application"
	delivery "github.com/example/hl7v2-message-router/internal/delivery/application"
	hl7app "github.com/example/hl7v2-message-router/internal/hl7/application"
	"github.com/example/hl7v2-message-router/internal/httpapi"
	"github.com/example/hl7v2-message-router/internal/mllp"
	routing "github.com/example/hl7v2-message-router/internal/routing/application"
	trace "github.com/example/hl7v2-message-router/internal/trace/application"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type handler struct {
	p *hl7app.Parser
	r *routing.Store
	d *delivery.Service
}

func (h handler) Handle(ctx context.Context, raw string) (string, error) {
	m, e := h.p.Parse(raw)
	if e != nil {
		return "", e
	}
	h.d.Deliver(ctx, m, h.r.Match(ctx, m))
	return "AA", nil
}
func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(1)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	p := hl7app.NewParser()
	rs := routing.NewStore()
	ds := delivery.New(delivery.MemoryConnector{})
	dl := dead.New()
	tr := trace.New()
	api := httpapi.New(p, rs, ds, dl, tr, logger)
	api.MiddlewareConfig = httpapi.DefaultMiddlewareConfig()
	api.MiddlewareConfig.AuthToken = cfg.AuthToken
	hs := &http.Server{Addr: cfg.HTTPAddr, Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second}
	ms := &mllp.Server{Addr: cfg.MLLPAddr, Handler: handler{p, rs, ds}, Logger: logger}
	go func() {
		logger.Info("http started", "addr", cfg.HTTPAddr)
		if e := hs.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			logger.Error("http", "error", e)
		}
	}()
	go func() {
		if e := ms.Start(); e != nil {
			logger.Error("mllp", "error", e)
		}
	}()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	hs.Shutdown(ctx)
	ms.Close()
}

package worker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"rule_engine/internal/config"
	"rule_engine/internal/metrics"
)

type App struct {
	server          *http.Server
	logger          *zap.Logger
	metrics         *metrics.PromMetrics
	shutdownTimeout time.Duration
}

func New(cfg *config.Config, logger *zap.Logger, m *metrics.PromMetrics) *App {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	addr := fmt.Sprintf("%s:%s", cfg.HTTPServer.Address, cfg.HTTPServer.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  cfg.HTTPServer.ReadTimeout,
		WriteTimeout: cfg.HTTPServer.WriteTimeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	return &App{
		server:          server,
		logger:          logger,
		metrics:         m,
		shutdownTimeout: cfg.HTTPServer.ShutdownTimeout,
	}
}

func (a *App) Start() {
	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Fatal("metrics server error", zap.Error(err))
		}
	}()
}

func (a *App) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, a.shutdownTimeout)
	defer cancel()
	return a.server.Shutdown(ctx)
}

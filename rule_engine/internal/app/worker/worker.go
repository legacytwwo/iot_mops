package worker

import (
	"context"

	"go.uber.org/zap"

	"rule_engine/internal/app/worker/liveness"
	"rule_engine/internal/config"
	"rule_engine/internal/metrics"
	"rule_engine/internal/repository/rabbit"
	"rule_engine/internal/service"
)

type Worker struct {
	cfg      *config.Config
	logger   *zap.Logger
	metrics  *metrics.PromMetrics
	consumer *rabbit.Client
	engine   *service.RuleEngine
	liveness *liveness.Worker
}

func NewWorker(cfg *config.Config, logger *zap.Logger, metrics *metrics.PromMetrics, consumer *rabbit.Client, engine *service.RuleEngine, livenessWorker *liveness.Worker) *Worker {
	return &Worker{
		cfg:      cfg,
		logger:   logger,
		metrics:  metrics,
		consumer: consumer,
		engine:   engine,
		liveness: livenessWorker,
	}
}

func (w *Worker) Start(ctx context.Context) error {
	processor := NewProcessor(w.engine, w.consumer, w.metrics, w.logger, w.cfg.Rabbit.RetryMaxAttempts)
	runner := NewRunner(w.consumer, processor, w.logger)
	if w.liveness != nil {
		go w.liveness.Start(ctx)
	}
	return runner.Run(ctx, w.cfg.Worker.Concurrency)
}

package worker

import (
	"context"
	"time"

	"go.uber.org/zap"

	"rule_engine/internal/metrics"
)

type LivenessWorker struct {
	interval time.Duration
	logger   *zap.Logger
	metrics  *metrics.PromMetrics
}

func NewLivenessWorker(interval time.Duration, logger *zap.Logger, metrics *metrics.PromMetrics) *LivenessWorker {
	return &LivenessWorker{
		interval: interval,
		logger:   logger,
		metrics:  metrics,
	}
}

func (w *LivenessWorker) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

package worker

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"rule_engine/internal/dto"
	"rule_engine/internal/metrics"
	"rule_engine/internal/repository/rabbit"
	"rule_engine/internal/service"
)

type Processor struct {
	engine      *service.RuleEngine
	rabbit      *rabbit.Client
	metrics     *metrics.PromMetrics
	logger      *zap.Logger
	maxAttempts int
}

func NewProcessor(engine *service.RuleEngine, rabbitClient *rabbit.Client, metrics *metrics.PromMetrics, logger *zap.Logger, maxAttempts int) *Processor {
	return &Processor{
		engine:      engine,
		rabbit:      rabbitClient,
		metrics:     metrics,
		logger:      logger,
		maxAttempts: maxAttempts,
	}
}

func (p *Processor) Handle(ctx context.Context, msg amqp.Delivery) error {
	start := time.Now()
	p.metrics.IncMessage("received")

	var envDTO dto.EnvelopeDTO
	if err := json.Unmarshal(msg.Body, &envDTO); err != nil {
		p.metrics.IncError("decode")
		p.logger.Error("decode message", zap.Error(err))
		_ = p.rabbit.PublishDLQ(ctx, msg, "decode_error")
		_ = msg.Ack(false)
		return nil
	}

	env, err := envDTO.ToEntity()
	if err != nil {
		p.metrics.IncError("map")
		p.logger.Error("map dto", zap.Error(err))
		_ = p.rabbit.PublishDLQ(ctx, msg, "map_error")
		_ = msg.Ack(false)
		return nil
	}

	alerts, err := p.engine.ProcessEnvelope(ctx, env)
	if err != nil {
		p.metrics.IncError("process")
		p.logger.Error("process envelope", zap.Error(err))
		p.engine.ReleaseMessageMark(ctx, env.MessageID)
		return p.retryOrDlq(ctx, msg)
	}

	if len(alerts) > 0 {
		if err := p.engine.SaveAlerts(ctx, alerts); err != nil {
			p.metrics.IncError("alerts_save")
			p.logger.Error("save alerts", zap.Error(err))
			p.engine.ReleaseMessageMark(ctx, env.MessageID)
			return p.retryOrDlq(ctx, msg)
		}
		for _, a := range alerts {
			p.metrics.IncAlert(a.RuleID, a.Severity)
		}
	}

	p.metrics.ObserveProcessDuration("total", time.Since(start))
	_ = msg.Ack(false)
	return nil
}

func (p *Processor) retryOrDlq(ctx context.Context, msg amqp.Delivery) error {
	attempt := rabbit.RetryCount(msg.Headers) + 1
	if attempt > p.maxAttempts {
		_ = p.rabbit.PublishDLQ(ctx, msg, "max_retries_exceeded")
		_ = msg.Ack(false)
		return nil
	}

	if pubErr := p.rabbit.PublishRetry(ctx, msg, attempt); pubErr != nil {
		return errors.New("retry publish failed")
	}
	_ = msg.Ack(false)
	return nil
}

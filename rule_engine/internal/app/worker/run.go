package worker

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"rule_engine/internal/repository/rabbit"
)

type Runner struct {
	consumer  *rabbit.Client
	processor *Processor
	logger    *zap.Logger
}

func NewRunner(consumer *rabbit.Client, processor *Processor, logger *zap.Logger) *Runner {
	return &Runner{consumer: consumer, processor: processor, logger: logger}
}

func (r *Runner) Run(ctx context.Context, workers int) error {
	deliveries, cleanup, err := r.consumer.Consume(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = cleanup() }()

	wg := sync.WaitGroup{}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for msg := range deliveries {
				if err := r.processor.Handle(ctx, msg); err != nil {
					r.logger.Error("handle message", zap.Error(err))
				}
			}
		}()
	}

	<-ctx.Done()
	wg.Wait()
	return nil
}

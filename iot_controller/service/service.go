package service

import (
	"context"
	"fmt"
	"iot_controller/entities"
	"iot_controller/metrics"
	"iot_controller/repository"
	"time"
)

type ServiceLayer interface {
	ProcessTelemetry(ctx context.Context, event *entities.Event) error
}

type telemetryService struct {
	metrics   *metrics.PromMetrics
	repo      repository.EventRepository
	publisher repository.EventPublisher
	redis     repository.LastSeenRepository
}

func NewService(
	m *metrics.PromMetrics,
	repo repository.EventRepository,
	pub repository.EventPublisher,
	redis repository.LastSeenRepository,
) ServiceLayer {
	return &telemetryService{
		metrics:   m,
		repo:      repo,
		publisher: pub,
		redis:     redis,
	}
}

func (s *telemetryService) ProcessTelemetry(ctx context.Context, event *entities.Event) error {
	if err := s.repo.SaveDevice(ctx, &event.Device); err != nil {
		s.metrics.IncMongoErrors()
		return fmt.Errorf("failed to save device metadata: %w", err)
	}

	if err := s.repo.SaveEvent(ctx, event); err != nil {
		s.metrics.IncMongoErrors()
		return fmt.Errorf("failed to save telemetry event: %w", err)
	}

	if err := s.publisher.Publish(ctx, event); err != nil {
		s.metrics.IncRabbitmqErrors()
		return fmt.Errorf("failed to publish event: %w", err)
	}

	if err := s.redis.SetLastSeen(ctx, event.Device.ID, time.Now()); err != nil {
		return fmt.Errorf("failed to save last seen timestamp: %w", err)
	}

	return nil
}

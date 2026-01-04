package service

import (
	"context"
	"fmt"
	"iot_controller/entities"
	"iot_controller/repository"
	"log"
)

type ServiceLayer interface {
	ProcessTelemetry(ctx context.Context, event *entities.Event) error
}

type telemetryService struct {
	repo      repository.EventRepository
	publisher repository.EventPublisher
}

func NewService(
	repo repository.EventRepository,
	pub repository.EventPublisher,
) ServiceLayer {
	return &telemetryService{
		repo:      repo,
		publisher: pub,
	}
}

func (s *telemetryService) ProcessTelemetry(ctx context.Context, event *entities.Event) error {
	if err := s.repo.SaveDevice(ctx, &event.Device); err != nil {
		return fmt.Errorf("failed to save device metadata: %w", err)
	}

	if err := s.repo.SaveEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to save telemetry event: %w", err)
	}

	if err := s.publisher.Publish(ctx, event); err != nil {
		log.Printf("Warning: failed to publish to rabbitmq: %v", err)
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

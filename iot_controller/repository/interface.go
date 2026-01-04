package repository

import (
	"context"
	"iot_controller/entities"
)

type EventRepository interface {
	SaveEvent(ctx context.Context, event *entities.Event) error
	SaveDevice(ctx context.Context, device *entities.Device) error
}

type EventPublisher interface {
	Publish(ctx context.Context, event *entities.Event) error
}

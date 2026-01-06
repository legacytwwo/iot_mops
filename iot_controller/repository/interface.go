package repository

import (
	"context"
	"iot_controller/entities"
	"time"
)

type EventRepository interface {
	SaveEvent(ctx context.Context, event *entities.Event) error
	SaveDevice(ctx context.Context, device *entities.Device) error
}

type EventPublisher interface {
	Publish(ctx context.Context, event *entities.Event) error
}

type LastSeenRepository interface {
	SetLastSeen(ctx context.Context, deviceID string, ts time.Time) error
}

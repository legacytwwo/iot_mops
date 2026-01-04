package rabbit

import (
	"context"
	"encoding/json"
	"fmt"
	"iot_controller/entities"
	"iot_controller/repository"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitPublisher struct {
	routingKey string
	ch         *amqp.Channel
}

func New(ch *amqp.Channel, routingKey string) repository.EventPublisher {
	return &rabbitPublisher{
		routingKey: routingKey,
		ch:         ch,
	}
}

func (r *rabbitPublisher) Publish(ctx context.Context, event *entities.Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = r.ch.PublishWithContext(ctx,
		"",
		r.routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    event.Timestamp,
			Body:         body,
		},
	)

	if err != nil {
		return fmt.Errorf("rabbit publish error: %w", err)
	}

	return nil
}

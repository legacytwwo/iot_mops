package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"iot_controller/entities"
	"iot_controller/service"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-playground/validator/v10"
)

type MQTTWorker struct {
	client    mqtt.Client
	validator *validator.Validate
	topic     string
	messageCh chan []byte
	service   service.ServiceLayer
}

func New(client mqtt.Client, topic string, service service.ServiceLayer) *MQTTWorker {
	return &MQTTWorker{
		client:    client,
		validator: validator.New(),
		topic:     topic,
		messageCh: make(chan []byte, 100),
		service:   service,
	}
}

func (w *MQTTWorker) StartConsumer() error {
	handler := func(client mqtt.Client, msg mqtt.Message) {
		payload := make([]byte, len(msg.Payload()))
		copy(payload, msg.Payload())
		w.messageCh <- payload
	}

	token := w.client.Subscribe(fmt.Sprintf("%s/+/telemetry", w.topic), 1, handler)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("subscribe failed: %w", token.Error())
	}

	log.Printf("MQTT Worker started. Listening on %s", w.topic)
	return nil
}

func (w *MQTTWorker) StartWorker() {
	go func() {
		for payload := range w.messageCh {
			if err := w.handleMessage(payload); err != nil {
				log.Printf("ERROR: Invalid JSON: %v", err)
			}
		}
	}()
}

func (w *MQTTWorker) handleMessage(payload []byte) error {
	var deviceEvent entities.Event
	if err := json.Unmarshal(payload, &deviceEvent); err != nil {
		return err
	}
	if err := w.validator.Struct(deviceEvent); err != nil {
		return err
	}
	if err := w.service.ProcessTelemetry(context.TODO(), &deviceEvent); err != nil {
		return err
	}
	return nil
}

package mqtt

import (
	"data_simulator/entities"
	"encoding/json"
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTClient struct {
	topicPrefix string
	client      mqtt.Client
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("Connected to MQTT Broker")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("Connection lost: %v", err)
}

func New(brokerURL, clientID, topicPrefix string) (*MQTTClient, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)
	opts.SetCleanSession(true)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &MQTTClient{
		topicPrefix: topicPrefix,
		client:      client,
	}, nil
}

func (c *MQTTClient) SendDeviceEvent(payload entities.Event, deviceID string) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	topic := fmt.Sprintf("%s/%s/telemetry", c.topicPrefix, deviceID)

	c.client.Publish(topic, 0, false, bytes)

	return nil
}

func (c *MQTTClient) GracefulStop() {
	c.client.Unsubscribe(c.topicPrefix)
	c.client.Disconnect(250)
}

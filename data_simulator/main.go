package main

import (
	"data_simulator/config"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Payload struct {
	DeviceID    string  `json:"device_id"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	Timestamp   int64   `json:"timestamp"`
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("Connected to MQTT Broker")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connection lost: %v", err)
}

func main() {
	cfg := config.LoadConfig()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.BrokerURL)
	opts.SetClientID(cfg.ClientID)
	opts.SetCleanSession(true)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to broker: %v", token.Error())
	}

	interval := time.Duration(float64(time.Second) / cfg.MsgRate)

	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	for i := 1; i <= cfg.DeviceCount; i++ {
		wg.Add(1)
		deviceID := fmt.Sprintf("device-%d", i)

		go func(id string) {
			defer wg.Done()
			generateData(client, id, cfg.TopicPrefix, interval, stopChan)
		}(deviceID)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("shutting down generator gracefully")
	close(stopChan)
	wg.Wait()
	client.Unsubscribe(cfg.TopicPrefix)
	client.Disconnect(250)
	log.Println("generator closed successfully")
}

func generateData(client mqtt.Client, deviceID, topicPrefix string, interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	topic := fmt.Sprintf("%s/%s", topicPrefix, deviceID)

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			payload := Payload{
				DeviceID:    deviceID,
				Temperature: 20.0 + rand.Float64()*15.0,
				Humidity:    30.0 + rand.Float64()*50.0,
				Timestamp:   time.Now().UnixMilli(),
			}

			bytes, err := json.Marshal(payload)
			if err != nil {
				log.Printf("JSON Error: %v", err)
				continue
			}

			token := client.Publish(topic, 0, false, bytes)

			go func() {
				token.Wait()
				if token.Error() != nil {
					log.Printf("Broker Error: %v", err)
				}
			}()
		}
	}
}

package main

import (
	"data_simulator/config"
	"data_simulator/generator"
	"data_simulator/repository/http"
	"data_simulator/repository/mqtt"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	cfg := config.LoadConfig()

	mqttClient, err := mqtt.New(cfg.MQTTConfig.BrokerURL, cfg.MQTTConfig.ClientID, cfg.MQTTConfig.TopicPrefix)
	if err != nil {
		log.Fatalf("Failed to connect to broker: %v", err)
	}

	httpClient := http.New(cfg.HTTPConfig.Timeout, cfg.HTTPConfig.BaseURL)

	interval := time.Duration(float64(time.Second) / cfg.MsgRate)

	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	for i := 1; i <= cfg.DeviceCount; i++ {
		wg.Add(1)

		go func(id string) {
			defer wg.Done()
			sendMessage(cfg.Mode, mqttClient, httpClient, id, interval, stopChan)
		}(fmt.Sprintf("%d", i))
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("shutting down generator gracefully")
	close(stopChan)
	wg.Wait()
	mqttClient.GracefulStop()
	log.Println("generator closed successfully")
}

func sendMessage(
	mode config.Mode,
	mqttClient *mqtt.MQTTClient,
	httpClient *http.HTTPClient,
	deviceID string,
	interval time.Duration,
	stop <-chan struct{},
) {
	useHTTP := mode == config.ModeHTTP || mode == config.ModeAll
	useMQTT := mode == config.ModeMQTT || mode == config.ModeAll

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			payload := generator.GenerateRandomEvent(deviceID)
			if useMQTT {
				err := mqttClient.SendDeviceEvent(payload, deviceID)
				if err != nil {
					log.Printf("MQTT Error: %v", err)
				}
			}
			if useHTTP {
				err := httpClient.SendDeviceEvent(payload)
				if err != nil {
					log.Printf("HTTP Error: %v", err)
				}
			}
		}
	}
}

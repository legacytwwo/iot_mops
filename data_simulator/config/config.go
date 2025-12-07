package config

import (
	"os"
	"strconv"
	"time"
)

type Mode string

const (
	ModeHTTP Mode = "http"
	ModeMQTT Mode = "mqtt"
	ModeAll  Mode = "all"
)

type Config struct {
	Mode        Mode
	DeviceCount int
	MsgRate     float64
	HTTPConfig  HTTPConfig
	MQTTConfig  MQTTConfig
}

type MQTTConfig struct {
	BrokerURL   string
	TopicPrefix string
	ClientID    string
}

type HTTPConfig struct {
	BaseURL string
	Timeout time.Duration
}

func LoadConfig() Config {
	return Config{
		Mode:        Mode(getEnvAsString("MODE", "all")),
		DeviceCount: getEnvAsInt("DEVICE_COUNT", 100),
		MsgRate:     getEnvAsFloat("MSG_RATE", 1.0),
		HTTPConfig: HTTPConfig{
			BaseURL: getEnvAsString("HTTP_BASE_URL", "http://localhost:8000"),
			Timeout: time.Duration(getEnvAsInt("HTTP_TIMEOUT", 5)) * time.Second,
		},
		MQTTConfig: MQTTConfig{
			TopicPrefix: getEnvAsString("TOPIC_PREFIX", "devices"),
			ClientID:    getEnvAsString("CLIENT_ID", "go-mqtt-client"),
			BrokerURL:   getEnvAsString("MQTT_BROKER", "tcp://localhost:1883"),
		},
	}
}

func getEnvAsString(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valueStr := getEnvAsString(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

func getEnvAsFloat(key string, defaultVal float64) float64 {
	valueStr := getEnvAsString(key, "")
	if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return value
	}
	return defaultVal
}

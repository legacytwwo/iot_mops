package config

import (
	"os"
	"strconv"
)

type Config struct {
	BrokerURL   string
	DeviceCount int
	MsgRate     float64
	TopicPrefix string
	ClientID    string
}

func LoadConfig() Config {
	return Config{
		BrokerURL:   getEnvAsString("MQTT_BROKER", "tcp://localhost:1883"),
		DeviceCount: getEnvAsInt("DEVICE_COUNT", 100),
		MsgRate:     getEnvAsFloat("MSG_RATE", 1.0),
		TopicPrefix: getEnvAsString("TOPIC_PREFIX", "iot/data"),
		ClientID:    getEnvAsString("CLIENT_ID", "go-mqtt-client"),
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

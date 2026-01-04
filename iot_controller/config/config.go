package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPServerConfig HTTPServerConfig
	MQTTConfig       MQTTConfig
	MongoDBConfig    MongoDBConfig
	RabbitMQConfig   RabbitMQConfig
}

type HTTPServerConfig struct {
	Address      string
	Port         string
	Timeout      time.Duration
	IdleTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type MQTTConfig struct {
	BrokerURL   string
	TopicPrefix string
	ClientID    string
}

type MongoDBConfig struct {
	URI              string
	Database         string
	EventCollection  string
	DeviceCollection string
	Timeout          time.Duration
}

type RabbitMQConfig struct {
	URL        string
	RoutingKey string
}

func LoadConfig() Config {
	return Config{
		HTTPServerConfig: HTTPServerConfig{
			Address:      getEnvAsString("HTTP_ADDRESS", "0.0.0.0"),
			Port:         getEnvAsString("HTTP_PORT", "8080"),
			Timeout:      time.Duration(getEnvAsInt("HTTP_TIMEOUT", 5)) * time.Second,
			IdleTimeout:  time.Duration(getEnvAsInt("HTTP_IDLE_TIMEOUT", 60)) * time.Second,
			ReadTimeout:  time.Duration(getEnvAsInt("HTTP_READ_TIMEOUT", 10)) * time.Second,
			WriteTimeout: time.Duration(getEnvAsInt("HTTP_WRITE_TIMEOUT", 10)) * time.Second,
		},
		MQTTConfig: MQTTConfig{
			BrokerURL:   getEnvAsString("MQTT_BROKER", "tcp://localhost:1883"),
			TopicPrefix: getEnvAsString("MQTT_TOPIC_PREFIX", "devices"),
			ClientID:    getEnvAsString("MQTT_CLIENT_ID", "go-mqtt-consumer"),
		},
		MongoDBConfig: MongoDBConfig{
			URI:              getEnvAsString("MONGO_URI", "mongodb://root:example@localhost:27017"),
			Database:         getEnvAsString("MONGO_DB", "iot_storage"),
			EventCollection:  getEnvAsString("MONGO_EVENT_COLLECTION", "telemetry"),
			DeviceCollection: getEnvAsString("MONGO_DEVICE_COLLECTION", "devices"),
			Timeout:          time.Duration(getEnvAsInt("MONGO_TIMEOUT", 10)) * time.Second,
		},
		RabbitMQConfig: RabbitMQConfig{
			URL:        getEnvAsString("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
			RoutingKey: getEnvAsString("RABBITMQ_ROUTING_KEY", "telemetry.envelopes"),
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

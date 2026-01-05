package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Environment string `env:"ENVIRONMENT" env-default:"local"`
	HTTPServer  HTTPServerConfig
	Logger      LoggerConfig
	Mongo       MongoConfig
	Redis       RedisConfig
	Rabbit      RabbitConfig
	Worker      WorkerConfig
	Liveness    LivenessConfig
}

type HTTPServerConfig struct {
	Address         string        `env:"HTTP_SERVER_ADDRESS" env-default:"0.0.0.0"`
	Port            string        `env:"HTTP_SERVER_PORT" env-default:"8080"`
	ReadTimeout     time.Duration `env:"HTTP_SERVER_READ_TIMEOUT" env-default:"5s"`
	WriteTimeout    time.Duration `env:"HTTP_SERVER_WRITE_TIMEOUT" env-default:"5s"`
	IdleTimeout     time.Duration `env:"HTTP_SERVER_IDLE_TIMEOUT" env-default:"60s"`
	ShutdownTimeout time.Duration `env:"HTTP_SERVER_SHUTDOWN_TIMEOUT" env-default:"10s"`
}

type LoggerConfig struct {
	Level     string `env:"LOGGER_LEVEL" env-default:"info"`
	Output    string `env:"LOGGER_OUTPUT" env-default:"stdout"`
	Formatter string `env:"LOGGER_FORMATTER" env-default:"json"`
}

type MongoConfig struct {
	URI            string        `env:"MONGO_URI" env-default:"mongodb://localhost:27017"`
	Database       string        `env:"MONGO_DB" env-default:"iot"`
	ConnectTimeout time.Duration `env:"MONGO_CONNECT_TIMEOUT" env-default:"5s"`
	MigrateOnStart bool          `env:"MONGO_MIGRATE_ON_START" env-default:"false"`
	MigrateLockTTL time.Duration `env:"MONGO_MIGRATE_LOCK_TTL" env-default:"1m"`
}

type RedisConfig struct {
	Addr         string        `env:"REDIS_ADDR" env-default:"localhost:6379"`
	Password     string        `env:"REDIS_PASSWORD" env-default:""`
	DB           int           `env:"REDIS_DB" env-default:"0"`
	PoolSize     int           `env:"REDIS_POOL_SIZE" env-default:"10"`
	MinIdleConns int           `env:"REDIS_MIN_IDLE_CONNS" env-default:"2"`
	DialTimeout  time.Duration `env:"REDIS_DIAL_TIMEOUT" env-default:"2s"`
}

type RabbitConfig struct {
	URL          string `env:"RABBIT_URL" env-default:"amqp://guest:guest@localhost:5672/"`
	Exchange     string `env:"RABBIT_EXCHANGE" env-default:"telemetry"`
	ExchangeType string `env:"RABBIT_EXCHANGE_TYPE" env-default:"direct"`
	Queue        string `env:"RABBIT_QUEUE" env-default:"telemetry.envelopes"`
	RoutingKey   string `env:"RABBIT_ROUTING_KEY" env-default:"telemetry.envelopes"`
	ConsumerTag  string `env:"RABBIT_CONSUMER_TAG" env-default:"rule-engine"`
	Prefetch     int    `env:"RABBIT_PREFETCH" env-default:"50"`

	RetryExchange     string `env:"RABBIT_RETRY_EXCHANGE" env-default:"telemetry.retry"`
	RetryExchangeType string `env:"RABBIT_RETRY_EXCHANGE_TYPE" env-default:"direct"`
	RetryQueue        string `env:"RABBIT_RETRY_QUEUE" env-default:"telemetry.envelopes.retry"`
	RetryRoutingKey   string `env:"RABBIT_RETRY_ROUTING_KEY" env-default:"telemetry.envelopes.retry"`

	DLXExchange     string `env:"RABBIT_DLX_EXCHANGE" env-default:"telemetry.dlq"`
	DLXExchangeType string `env:"RABBIT_DLX_EXCHANGE_TYPE" env-default:"direct"`
	DLQQueue        string `env:"RABBIT_DLQ_QUEUE" env-default:"telemetry.envelopes.dlq"`
	DLQRoutingKey   string `env:"RABBIT_DLQ_ROUTING_KEY" env-default:"telemetry.envelopes.dlq"`

	RetryBaseDelay   time.Duration `env:"RABBIT_RETRY_BASE_DELAY" env-default:"5s"`
	RetryMaxDelay    time.Duration `env:"RABBIT_RETRY_MAX_DELAY" env-default:"60s"`
	RetryMaxAttempts int           `env:"RABBIT_RETRY_MAX_ATTEMPTS" env-default:"5"`
	RetryJitter      float64       `env:"RABBIT_RETRY_JITTER" env-default:"0.2"`
}

type WorkerConfig struct {
	Concurrency     int           `env:"WORKER_CONCURRENCY" env-default:"1"`
	ShutdownTimeout time.Duration `env:"WORKER_SHUTDOWN_TIMEOUT" env-default:"10s"`
}

type LivenessConfig struct {
	Interval time.Duration `env:"LIVENESS_INTERVAL" env-default:"5s"`
}

func GetConfig() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return &cfg, nil
}

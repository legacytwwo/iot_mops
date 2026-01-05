package box

import (
	"context"
	"fmt"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"rule_engine/internal/config"
	"rule_engine/internal/metrics"
	"rule_engine/internal/repository/rabbit"
)

type Env struct {
	Config     *config.Config
	Logger     *zap.Logger
	Metrics    *metrics.PromMetrics
	Mongo      *mongo.Client
	Redis      *redis.Client
	RabbitConn *amqp.Connection
	Rabbit     *rabbit.Client
}

func New() (Env, error) {
	_ = godotenv.Load()

	cfg, err := config.GetConfig()
	if err != nil {
		return Env{}, err
	}

	lg, err := provideLogger(cfg)
	if err != nil {
		return Env{}, err
	}

	m := metrics.NewServiceMetrics(lg)

	mongoClient, err := provideMongo(cfg)
	if err != nil {
		return Env{}, err
	}

	redisClient, err := provideRedis(cfg)
	if err != nil {
		return Env{}, err
	}

	rabbitConn, err := provideRabbit(cfg)
	if err != nil {
		return Env{}, err
	}

	rabbitClient := rabbit.New(rabbitConn, cfg.Rabbit, lg)
	if err := rabbitClient.SetupTopology(); err != nil {
		return Env{}, err
	}

	return Env{
		Config:     cfg,
		Logger:     lg,
		Metrics:    m,
		Mongo:      mongoClient,
		Redis:      redisClient,
		RabbitConn: rabbitConn,
		Rabbit:     rabbitClient,
	}, nil
}

func provideLogger(cfg *config.Config) (*zap.Logger, error) {
	lvl := zap.InfoLevel
	if cfg.Logger.Level == "debug" {
		lvl = zap.DebugLevel
	}

	logCfg := zap.Config{
		Encoding:    cfg.Logger.Formatter,
		Level:       zap.NewAtomicLevelAt(lvl),
		OutputPaths: []string{cfg.Logger.Output},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:  "msg",
			LevelKey:    "level",
			TimeKey:     "ts",
			EncodeTime:  zapcore.ISO8601TimeEncoder,
			EncodeLevel: zapcore.LowercaseLevelEncoder,
		},
	}

	logger, err := logCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}
	return logger, nil
}

func provideMongo(cfg *config.Config) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Mongo.ConnectTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("mongo ping: %w", err)
	}
	return client, nil
}

func provideRedis(cfg *config.Config) (*redis.Client, error) {
	opts := &redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
	}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return client, nil
}

func provideRabbit(cfg *config.Config) (*amqp.Connection, error) {
	conn, err := amqp.Dial(cfg.Rabbit.URL)
	if err != nil {
		return nil, fmt.Errorf("rabbit dial: %w", err)
	}
	return conn, nil
}

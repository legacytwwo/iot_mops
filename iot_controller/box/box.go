package box

import (
	"context"
	"fmt"
	"iot_controller/config"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type Box struct {
	Config        config.Config
	RabbitConn    *amqp.Connection
	RabbitCh      *amqp.Channel
	MQTTReader    mqtt.Client
	MongoDBClient *mongo.Database
	Redis         *redis.Client
}

func New() (*Box, error) {
	cfg := config.LoadConfig()

	mongoClient, err := initMongoDB(cfg.MongoDBConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to init mongodb: %w", err)
	}

	rabbitConn, rabbitCh, err := initRabbitMQ(cfg.RabbitMQConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to init rabbitmq: %w", err)
	}

	mqttclient, err := initMQTT(cfg.MQTTConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to init mqtt: %w", err)
	}

	redisClient, err := provideRedis(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init redis: %w", err)
	}

	return &Box{
		Config:        cfg,
		RabbitConn:    rabbitConn,
		RabbitCh:      rabbitCh,
		MongoDBClient: mongoClient,
		MQTTReader:    mqttclient,
		Redis:         redisClient,
	}, nil
}

func initMongoDB(cfg config.MongoDBConfig) (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	clientOpts := options.Client().ApplyURI(cfg.URI)
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	return client.Database(cfg.Database), nil
}

func initRabbitMQ(cfg config.RabbitMQConfig) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	return conn, ch, nil
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("Connected to MQTT Broker")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("Connection lost: %v", err)
}

func initMQTT(cfg config.MQTTConfig) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.BrokerURL)
	opts.SetClientID(cfg.ClientID)
	opts.SetCleanSession(true)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return client, nil
}

func provideRedis(cfg config.Config) (*redis.Client, error) {
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
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return client, nil
}

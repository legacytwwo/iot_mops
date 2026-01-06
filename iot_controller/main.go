package main

import (
	"context"
	httpApp "iot_controller/app/http"
	"iot_controller/app/http/handlers"
	mqttApp "iot_controller/app/mqtt"
	"iot_controller/box"
	"iot_controller/metrics"
	"iot_controller/repository/mongo"
	rabbit "iot_controller/repository/rabbitmq"
	"iot_controller/repository/redis"
	"iot_controller/service"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	envBox, err := box.New()
	if err != nil {
		log.Fatalf("failed to init box, %v", err)
	}

	mongoRepository := mongo.New(
		envBox.MongoDBClient,
		envBox.Config.MongoDBConfig.EventCollection,
		envBox.Config.MongoDBConfig.DeviceCollection,
	)
	rabbitRepository := rabbit.New(envBox.RabbitCh, envBox.Config.RabbitMQConfig.RoutingKey)
	redisRepository := redis.New(envBox.Redis)

	m := metrics.NewServiceMetrics()

	service := service.NewService(m, mongoRepository, rabbitRepository, redisRepository)

	h := handlers.New(m, service)

	httpServerSettings := &http.Server{
		Handler:      h.Router,
		Addr:         net.JoinHostPort(envBox.Config.HTTPServerConfig.Address, envBox.Config.HTTPServerConfig.Port),
		ReadTimeout:  envBox.Config.HTTPServerConfig.ReadTimeout,
		WriteTimeout: envBox.Config.HTTPServerConfig.WriteTimeout,
		IdleTimeout:  envBox.Config.HTTPServerConfig.IdleTimeout,
		ConnState: func(c net.Conn, cs http.ConnState) {
			switch cs {
			case http.StateNew:
				m.IncActiveConnections()
			case http.StateHijacked, http.StateClosed:
				m.DecActiveConnections()
			}
		},
	}

	httpServer := httpApp.New(httpServerSettings)
	mqttServer := mqttApp.New(m, envBox.MQTTReader, envBox.Config.MQTTConfig.TopicPrefix, service)

	httpServer.Run()
	mqttServer.StartConsumer()
	mqttServer.StartWorker()

	log.Printf(
		"listening on %s",
		net.JoinHostPort(envBox.Config.HTTPServerConfig.Address, envBox.Config.HTTPServerConfig.Port),
	)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("shutting down controller gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err = httpServer.GracefullyStop(shutdownCtx); err != nil {
		log.Fatalf("failed to shutdown http server: %v", err)
	}

	if err = envBox.MongoDBClient.Client().Disconnect(shutdownCtx); err != nil {
		log.Fatalf("failed to shutdown mongo client: %v", err)
	}

	if err = envBox.RabbitConn.Close(); err != nil {
		log.Fatalf("failed to shutdown rabbit client: %v", err)
	}

	if err := envBox.Redis.Close(); err != nil {
		log.Fatalf("failed to shutdown redis client: %v", err)
	}

	log.Println("controller closed successfully")
}

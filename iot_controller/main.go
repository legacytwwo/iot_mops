package main

import (
	httpApp "iot_controller/app/http"
	"iot_controller/app/http/handlers"
	mqttApp "iot_controller/app/mqtt"
	"iot_controller/box"
	"iot_controller/metrics"
	"iot_controller/repository/mongo"
	rabbit "iot_controller/repository/rabbitmq"
	"iot_controller/service"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	service := service.NewService(mongoRepository, rabbitRepository)

	m := metrics.NewServiceMetrics()

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
	mqttServer := mqttApp.New(envBox.MQTTReader, envBox.Config.MQTTConfig.TopicPrefix, service)

	httpServer.Run()
	mqttServer.StartConsumer()
	mqttServer.StartWorker()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("shutting down generator gracefully")
	log.Println("generator closed successfully")
}

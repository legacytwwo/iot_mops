package handlers

import (
	"iot_controller/metrics"
	"iot_controller/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type httpHandlers struct {
	validator *validator.Validate
	Router    *chi.Mux
	service   service.ServiceLayer
}

func New(m *metrics.PromMetrics, service service.ServiceLayer) *httpHandlers {
	h := &httpHandlers{
		validator: validator.New(),
		service:   service,
	}
	router := h.setRouter(m)
	h.Router = router
	return h
}

func (h *httpHandlers) setRouter(m *metrics.PromMetrics) *chi.Mux {
	router := chi.NewRouter()

	router.Use(
		middleware.Heartbeat("/ping"),
	)

	router.Handle("/metrics", promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{EnableOpenMetrics: true}))

	router.Post("/telemetry", h.saveTelemetry())

	return router
}

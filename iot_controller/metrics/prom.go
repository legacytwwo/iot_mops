package metrics

import (
	"reflect"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

const prefix = "iot_service_"

type metrics struct {
	MessagesTotal     *prometheus.CounterVec
	ValidationErrors  prometheus.Counter
	MongoErrors       prometheus.Counter
	RabbitmqErrors    prometheus.Counter
	ProcessDuration   *prometheus.HistogramVec
	ActiveConnections prometheus.Gauge
}

type PromMetrics struct {
	metrics    *metrics
	Registry   *prometheus.Registry
	registerer prometheus.Registerer
}

func NewServiceMetrics() *PromMetrics {
	registry := prometheus.NewRegistry()
	wrappedReg := prometheus.WrapRegistererWithPrefix(prefix, registry)

	metrics := &metrics{
		ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "active_connections",
		}),
		MessagesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "messages_total",
			Help: "total messages received from queue",
		}, []string{"status"}),
		ValidationErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "validation_errors",
			Help: "validation_errors",
		}),
		MongoErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "mongo_errors",
			Help: "mongo_errors",
		}),
		RabbitmqErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rabbit_errors",
			Help: "rabbit_errors",
		}),
		ProcessDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "process_duration_ms",
			Help:    "processing duration per envelope",
			Buckets: prometheus.ExponentialBucketsRange(1, 10_000, 12),
		}, []string{"stage"}),
	}

	m := &PromMetrics{
		metrics:    metrics,
		Registry:   registry,
		registerer: wrappedReg,
	}

	m.registerAllMetrics()

	return m
}

func (m *PromMetrics) registerAllMetrics() {
	m.registerer.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	v := reflect.ValueOf(m.metrics).Elem()

	for i := range v.NumField() {
		field := v.Field(i)
		if metric, ok := field.Interface().(prometheus.Collector); ok {
			m.registerer.MustRegister(metric)
		}
	}
}

func (m *PromMetrics) IncActiveConnections() {
	m.metrics.ActiveConnections.Inc()
}

func (m *PromMetrics) DecActiveConnections() {
	m.metrics.ActiveConnections.Dec()
}

func (m *PromMetrics) IncMessage(status string) {
	m.metrics.MessagesTotal.WithLabelValues(status).Inc()
}

func (m *PromMetrics) ObserveProcessDuration(stage string, d time.Duration) {
	m.metrics.ProcessDuration.WithLabelValues(stage).Observe(float64(d.Milliseconds()))
}

func (m *PromMetrics) IncValidationErrors() {
	m.metrics.ValidationErrors.Inc()
}

func (m *PromMetrics) IncMongoErrors() {
	m.metrics.MongoErrors.Inc()
}

func (m *PromMetrics) IncRabbitmqErrors() {
	m.metrics.RabbitmqErrors.Inc()
}

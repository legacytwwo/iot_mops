package metrics

import (
	"reflect"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

const prefix = "iot_service_"

type metrics struct {
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

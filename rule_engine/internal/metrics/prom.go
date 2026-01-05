package metrics

import (
	"reflect"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.uber.org/zap"
)

type metrics struct {
	MessagesTotal   *prometheus.CounterVec
	AlertsTotal     *prometheus.CounterVec
	ErrorsTotal     *prometheus.CounterVec
	ProcessDuration *prometheus.HistogramVec
	LivenessChecks  prometheus.Counter
	LivenessAlerts  prometheus.Counter
	LivenessErrors  *prometheus.CounterVec
}

type PromMetrics struct {
	metrics    *metrics
	lg         *zap.Logger
	Registry   *prometheus.Registry
	registerer prometheus.Registerer
}

const subSystemPrefix = "rule_engine_"

func NewServiceMetrics(lg *zap.Logger) *PromMetrics {
	registry := prometheus.NewRegistry()
	wrappedReg := prometheus.WrapRegistererWithPrefix(subSystemPrefix, registry)

	mSet := &metrics{
		MessagesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "messages_total",
			Help: "total messages received from queue",
		}, []string{"status"}),
		AlertsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "alerts_total",
			Help: "total alerts created",
		}, []string{"rule_id", "severity"}),
		ErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "errors_total",
			Help: "total processing errors",
		}, []string{"stage"}),
		ProcessDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "process_duration_ms",
			Help:    "processing duration per envelope",
			Buckets: prometheus.ExponentialBucketsRange(1, 10_000, 12),
		}, []string{"stage"}),
		LivenessChecks: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "liveness_checks_total",
			Help: "total liveness checks",
		}),
		LivenessAlerts: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "liveness_alerts_total",
			Help: "total liveness alerts",
		}),
		LivenessErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "liveness_errors_total",
			Help: "total liveness errors",
		}, []string{"stage"}),
	}

	m := &PromMetrics{
		metrics:    mSet,
		lg:         lg,
		Registry:   registry,
		registerer: wrappedReg,
	}

	m.registerAll()
	return m
}

func (m *PromMetrics) registerAll() {
	m.registerer.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	v := reflect.ValueOf(m.metrics).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if metric, ok := field.Interface().(prometheus.Collector); ok {
			m.registerer.MustRegister(metric)
		}
	}
}

func (m *PromMetrics) Unregister() {
	v := reflect.ValueOf(m.metrics).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if metric, ok := field.Interface().(prometheus.Collector); ok {
			m.registerer.Unregister(metric)
		}
	}
}

func (m *PromMetrics) IncMessage(status string) {
	m.metrics.MessagesTotal.WithLabelValues(status).Inc()
}

func (m *PromMetrics) IncAlert(ruleID, severity string) {
	m.metrics.AlertsTotal.WithLabelValues(ruleID, severity).Inc()
}

func (m *PromMetrics) IncError(stage string) {
	m.metrics.ErrorsTotal.WithLabelValues(stage).Inc()
}

func (m *PromMetrics) ObserveProcessDuration(stage string, d time.Duration) {
	m.metrics.ProcessDuration.WithLabelValues(stage).Observe(float64(d.Milliseconds()))
}

func (m *PromMetrics) IncLivenessCheck() {
	m.metrics.LivenessChecks.Inc()
}

func (m *PromMetrics) IncLivenessAlert(count int) {
	for i := 0; i < count; i++ {
		m.metrics.LivenessAlerts.Inc()
	}
}

func (m *PromMetrics) IncLivenessError(stage string) {
	m.metrics.LivenessErrors.WithLabelValues(stage).Inc()
}

package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/push"
)

var (
	logEntriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "log_entries_total",
			Help: "Total number of log entries written per scene",
		},
		[]string{"scene"},
	)

	logLevelTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "log_level_total",
			Help: "Total number of log entries by level",
		},
		[]string{"level"},
	)

	mqPushTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mq_push_total",
			Help: "Total number of MQ push operations",
		},
		[]string{"status"},
	)

	fsyncLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fsync_latency_seconds",
			Help:    "Latency of fsync operations",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"mode"},
	)

	bufferUsageRatio = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "buffer_usage_ratio",
			Help: "Current buffer usage ratio per scene",
		},
		[]string{"scene"},
	)

	vitalP99LatencyMs = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "vital_p99_latency_ms",
			Help: "P99 latency for Vital scene in milliseconds",
		},
	)
)

func InitMetrics() {
	prometheus.MustRegister(logEntriesTotal)
	prometheus.MustRegister(logLevelTotal)
	prometheus.MustRegister(mqPushTotal)
	prometheus.MustRegister(fsyncLatency)
	prometheus.MustRegister(bufferUsageRatio)
	prometheus.MustRegister(vitalP99LatencyMs)
}

func Handler() http.Handler {
	return promhttp.Handler()
}

func RecordLogEntry(scene, level string) {
	logEntriesTotal.WithLabelValues(scene).Inc()
	logLevelTotal.WithLabelValues(level).Inc()
}

func RecordMQPush(success bool) {
	status := "success"
	if !success {
		status = "error"
	}
	mqPushTotal.WithLabelValues(status).Inc()
}

func RecordFsyncLatency(seconds float64, mode string) {
	fsyncLatency.WithLabelValues(mode).Observe(seconds)
}

func SetBufferUsage(scene string, ratio float64) {
	bufferUsageRatio.WithLabelValues(scene).Set(ratio)
}

func SetVitalP99Latency(ms float64) {
	vitalP99LatencyMs.Set(ms)
}

type Pusher struct {
	addr   string
	pusher *push.Pusher
}

func NewPusher(addr string, jobName string) *Pusher {
	return &Pusher{
		addr:   addr,
		pusher: push.New(addr, jobName),
	}
}

func (p *Pusher) Add() *push.Pusher {
	return p.pusher
}

func (p *Pusher) Push() error {
	return p.pusher.Push()
}

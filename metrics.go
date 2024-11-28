package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type PrometheusMetrics struct {
	cpuUsage    prometheus.Gauge
	memoryUsage prometheus.Gauge
	diskIo      *prometheus.CounterVec
	networkIo   *prometheus.CounterVec

	handler http.Handler
}

func NewPrometheusMetricsManager() *PrometheusMetrics {
	metricsManager := &PrometheusMetrics{
		cpuUsage:    prometheus.NewGauge(prometheus.GaugeOpts{Name: "cpu_usage", Help: "CPU usage"}),
		memoryUsage: prometheus.NewGauge(prometheus.GaugeOpts{Name: "memory_usage", Help: "Memory usage"}),
		diskIo:      prometheus.NewCounterVec(prometheus.CounterOpts{Name: "disk_io"}, []string{"device", "io"}),
		networkIo:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: "network_io"}, []string{"interface", "io"}),
		handler:     promhttp.Handler(),
	}
	prometheus.MustRegister(metricsManager.cpuUsage, metricsManager.memoryUsage,
		metricsManager.diskIo, metricsManager.networkIo)
	return metricsManager
}

func (m *PrometheusMetrics) UpdateCPUUsage(usage float64) {
	m.cpuUsage.Set(usage)
}

func (m *PrometheusMetrics) UpdateMemoryUsage(usage float64) {
	m.memoryUsage.Set(usage)
}

func (m *PrometheusMetrics) UpdateDiskIo(diskName string, ioStr string, value float64) {
	m.diskIo.WithLabelValues(diskName, ioStr).Add(value)
}

func (m *PrometheusMetrics) UpdateNetworkIo(networkName string, ioStr string, value float64) {
	m.networkIo.WithLabelValues(networkName, ioStr).Add(value)
}

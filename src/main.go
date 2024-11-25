package main

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"time"
)

type Alert struct {
	Threshold int
	Readings  int
}

type Config struct {
	Host     string
	Interval int

	PrometheusHost string
	PrometheusPort string
	Alert          Alert
}

const MB = 1024 * 1024

func main() {
	m := Metrics{
		DiskCounters:    make(map[string]DiskStats),
		NetworkCounters: make(map[string]NetworkStats),
	}

	configFile, err := os.Open("config.json")
	if err != nil {
		log.Fatal(err)
	}
	defer configFile.Close()
	var config Config
	err = json.NewDecoder(configFile).Decode(&config)
	if err != nil {
		log.Fatal(err)
	}

	cpuMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "CPUStats",
	})
	prometheus.MustRegister(cpuMetric)

	memMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "MemoryStats",
	})
	prometheus.MustRegister(memMetric)

	diskMetrics := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "DiskStats",
	}, []string{"device", "io"})
	prometheus.MustRegister(diskMetrics)

	networkMetrics := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "NetworkStats",
	}, []string{"interface", "io"})
	prometheus.MustRegister(networkMetrics)

	var readingCounter int
	var total float64

	ticker := time.NewTicker(time.Duration(config.Interval) * time.Second)
	go func() {
		for range ticker.C {
			m.Collect()

			cpuMetric.Set(float64(m.CPUUsage))
			total = total + float64(m.CPUUsage)
			if readingCounter >= config.Alert.Readings {
				avg := total / float64(readingCounter)
				if avg >= float64(config.Alert.Threshold) {
					generateAlert(fmt.Sprintf("CPU Usage remained above %d for last %d seconds\n",
						config.Alert.Threshold, config.Interval*readingCounter))
				}
				readingCounter, total = 0, 0
			}

			memMetric.Set(float64(m.MemoryUsage))

			for diskName, stats := range m.DiskCounters {
				diskMetrics.WithLabelValues(diskName, "read").Add(float64(stats.ReadBytes / MB))
				diskMetrics.WithLabelValues(diskName, "write").Add(float64(stats.WriteBytes / MB))
			}

			for interfaceName, stats := range m.NetworkCounters {
				networkMetrics.WithLabelValues(interfaceName, "received").Add(float64(stats.BytesRecv / MB))
				networkMetrics.WithLabelValues(interfaceName, "sent").Add(float64(stats.BytesSent / MB))
			}
			readingCounter++
		}
	}()

	http.HandleFunc("/query", handleQuery(config))
	http.HandleFunc("/avg", handleAvg(config))
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8080", nil)
}

func generateAlert(alertStr string) {
	alertFile, err := os.OpenFile("alert.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Println("error writing alert to file", err)
		return
	}
	defer alertFile.Close()

	alertFile.WriteString(alertStr)
}

// handleQuery /query endpoint that takes query, start time, end time and step parameter
// example: http://localhost:8080/query?query=CPUStats&end=2024-11-25T06:03:32Z&start=2024-11-25T05:59:32Z&step=15s
func handleQuery(config Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")
		step := r.URL.Query().Get("step")

		pc := PrometheusClient{
			Hostname: config.PrometheusHost,
			Port:     config.PrometheusPort,
		}

		response, err := pc.QueryStatsForTimeRange(query, start, end, step)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(response.Data.Result)
	}
}

func handleAvg(config Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")
		step := r.URL.Query().Get("step")

		pc := PrometheusClient{
			Hostname: config.PrometheusHost,
			Port:     config.PrometheusPort,
		}
		avg, err := pc.AvgUsage(query, start, end, step)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		type Response struct {
			Metric string
			Avg    float64
		}
		var response Response
		response.Metric = query
		response.Avg = avg
		json.NewEncoder(w).Encode(response)
	}
}

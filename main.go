package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type MetricsManager interface {
	UpdateCPUUsage(float64)
	UpdateMemoryUsage(float64)
	UpdateDiskIo(string, string, float64)    //args: diskName, ioStr, value
	UpdateNetworkIo(string, string, float64) // args: interfaceName, ioStr, value
}

type Alert struct {
	Threshold int
	Readings  int
	FileName  string
}

// WriteAlert writes alert to the file
func (a *Alert) WriteAlert(alert string) {
	alertFile, err := os.OpenFile(a.FileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Println("error writing alert to file", err)
		return
	}
	defer alertFile.Close()
	alertFile.WriteString(alert)
}

type Config struct {
	ApiServerPort  string
	Interval       int
	PrometheusHost string
	PrometheusPort string
	Alert          Alert
}

const MB = 1024 * 1024

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Usage: monitoring path/to/config.json")
		return
	}

	configFile, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer configFile.Close()
	var config Config
	err = json.NewDecoder(configFile).Decode(&config)
	if err != nil {
		log.Fatal(err)
	}

	metricsManager := NewPrometheusMetricsManager()

	ticker := time.NewTicker(time.Duration(config.Interval) * time.Second)
	go func() {
		var readingCounter int
		var total float64

		for range ticker.C {
			cpuUsage, err := RefreshCpuUsage(metricsManager)
			if err != nil {
				log.Println(err)
			}

			//generate alert for cpuUsage
			total = total + cpuUsage
			if readingCounter >= config.Alert.Readings {
				avg := total / float64(readingCounter)
				if avg >= float64(config.Alert.Threshold) {
					alertStr := fmt.Sprintf("CPU Usage remained above %d for last %d seconds\n",
						config.Alert.Threshold, config.Interval*readingCounter)
					config.Alert.WriteAlert(alertStr)
				}
				readingCounter, total = 0, 0
			}

			err = RefreshMemoryUsage(metricsManager)
			if err != nil {
				log.Println(err)
			}

			err = RefreshDiskCounters(metricsManager)
			if err != nil {
				log.Println(err)
			}

			err = RefreshNetworkCounters(metricsManager)
			if err != nil {
				log.Println(err)
			}

			readingCounter++
		}
	}()
	log.Printf("Started monitoring system metrics at interval of %d seconds \n", config.Interval)
	log.Println("API server available at port:", config.ApiServerPort)
	http.HandleFunc("/query", handleQuery(config))
	http.HandleFunc("/avg", handleAvg(config))
	http.Handle("/metrics", metricsManager.handler)
	http.ListenAndServe(":"+config.ApiServerPort, nil)
}

// handleQuery /query endpoint that takes query, start time, end time and step parameter
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

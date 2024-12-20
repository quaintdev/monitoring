# Metrics Collector

This application collects metrics of system like CPU Usage, Memory Usage, disk i/o and network i/o.
It then displays these metrics using Grafana. It supports following features

## Features
- Collect metrics at configured interval
- Store collected metrics in prometheus
- REST API to view stored metrics
    - Query metrics over given time range
    - Query metric aggregate over given time range
- Stores configured alerts in `alert.txt`
- View stored metrics using grafana

## Note
- Currently, application only supports metric collection from Linux system

## Installation
### Pre-requisites
1. Create volumes for prometheus and grafana
    ```shell
    podman volume create prometheus-config
    podman volume create prometheus-data
    podman volume create grafana-data
    ```
2. Below commands will create prometheus and grafana containers that application uses to store and view metrics
    ```shell
    podman run -d --name prometheus --network host \
    -v prometheus-config:/etc/prometheus \
    -v prometheus-data:/prometheus \
    prom/prometheus:latest
    
    podman run -d --name grafana --network host \
    -v grafana-data:/var/lib/grafana \
    -e GF_SECURITY_ADMIN_USER=admin \
    -e GF_SECURITY_ADMIN_PASSWORD=admin \
    grafana/grafana-oss:latest 
    ```
   **Note**: You will have to update `/etc/prometheus/prometheus.yml` within the `prometheus-config` volume.
   You can use `podman volume inspect` to determine where the `prometheus-config` volume is mounted.
   Your `prometheus.yml` should look like below
   ```yaml
   global:
      scrape_interval: 3s
   scrape_configs:
      - job_name: "prometheus"
        static_configs:
         - targets: ["localhost:8080"] # the port here should be same as that of API endpoint
   ```
## Install application
1. The application can be installed using below command
   ```shell
   go install github.com/quaintdev/monitoring@latest
   ```
2. Before execution ensure that you have a config file ready whose structure should look like below
    ```json
    {
      "apiServerPort": "8080",
      "interval": 5,
      "prometheusHost": "localhost",
      "prometheusPort": "9090",
      "alert": {
        "threshold": 80,
        "readings": 60
      }
    }
   ```
   - `apiServerPort` - API server port
   - `interval` - polling interval in seconds  
   - `prometheusHost` & `prometheusPort` - should be set to prometheus endpoint config  
   - `alert` specifies CPU `threshold` above which the alert should be generated. Here `readings` specifies the number of reading to be used for calculating average. Alerts are saved to `alert.txt`
3. Once you are ready with `config.json` you can run application using below command
    ```shell
    monitoring path/to/config.json
    ```
## Grafana

![Dashboards - Grafana](https://github.com/user-attachments/assets/741fbb54-dbb8-4be1-9dd4-728aefdf6ef3)

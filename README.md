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

## Installation
1. 

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type PrometheusClient struct {
	Hostname string
	Port     string
}

type Result struct {
	Values [][]interface{}
}

type Data struct {
	Result []Result
}

type PrometheusResponse struct {
	Status string `json:"status"`
	Data   Data
}

func (c *PrometheusClient) GetHostUrl() string {
	return c.Hostname + ":" + c.Port
}

func (c *PrometheusClient) QueryStatsForTimeRange(metric, startTime, endTime, step string) (*PrometheusResponse, error) {
	reqUrl := fmt.Sprintf("http://%s/api/v1/query_range?query=%s&start=%s&end=%s&step=%s",
		c.GetHostUrl(), metric, startTime, endTime, step)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return nil, err
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response *PrometheusResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *PrometheusClient) AvgUsage(metric, startTime, endTime, step string) (float64, error) {
	response, err := c.QueryStatsForTimeRange(metric, startTime, endTime, step)
	if err != nil {
		return 0, err
	}
	var total, counter float64
	for _, value := range response.Data.Result[0].Values {
		if len(value) >= 1 {
			i, err := strconv.Atoi(value[1].(string))
			if err != nil {
				return 0, err
			}
			total += float64(i)
			counter++
		}
	}
	avg := total / counter
	return avg, nil
}

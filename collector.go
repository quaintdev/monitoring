package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func RefreshMemoryUsage(manager MetricsManager) error {
	cmd := exec.Command("free", "--line", "--mega")
	cmdOutputByte, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("could not execute free: %s", err)
	}

	pattern := `(\w+)\s+(\d+)`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(string(cmdOutputByte), -1)
	if matches == nil {
		return fmt.Errorf("no matches found in input")
	}
	result := make(map[string]int)
	for _, match := range matches {
		key := match[1]
		value, err := strconv.Atoi(match[2])
		if err != nil {
			return fmt.Errorf("error converting value for key %s: %w", key, err)
		}
		result[key] = value
	}

	freeMem := result["CachUse"] + result["MemFree"]
	memoryUsage := result["MemUse"] * 100 / (freeMem + result["MemUse"])
	manager.UpdateMemoryUsage(float64(memoryUsage))
	return nil
}

var prevCPUReading [10]int

func RefreshCpuUsage(metricsManager MetricsManager) (float64, error) {
	statsFile, err := os.Open("/proc/stat")
	if err != nil {
		return 0, fmt.Errorf("could not open /proc/diskstats: %s", err)
	}
	defer statsFile.Close()

	scanner := bufio.NewScanner(statsFile)
	scanner.Scan()
	line := scanner.Text()
	line = strings.TrimSpace(line)
	fieldRow := strings.Fields(line)
	var idle, total int

	for k, field := range fieldRow {
		if k == 0 {
			continue
		}
		fieldValue, err := strconv.Atoi(field)
		if err != nil {
			return 0, fmt.Errorf("could not convert total value for field %s", field)
		}
		if k == 4 {
			idle = fieldValue - prevCPUReading[k-1]
		}
		total = total + fieldValue - prevCPUReading[k-1]
		prevCPUReading[k-1] = fieldValue
	}
	var cpuUsage float64
	if total > 0 {
		cpuUsage = 100 - float64(idle)*100/float64(total)
		metricsManager.UpdateCPUUsage(cpuUsage)
	}
	return cpuUsage, nil
}

const sectorSize = 512

func RefreshDiskCounters(manager MetricsManager) error {
	statsFile, err := os.Open("/proc/diskstats")
	if err != nil {
		return fmt.Errorf("could not open /proc/diskstats: %s", err)
	}
	defer statsFile.Close()

	scanner := bufio.NewScanner(statsFile)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		fieldRow := strings.Fields(line)
		sectorsRead, err := strconv.Atoi(fieldRow[5])
		if err != nil {
			log.Printf("could not parse disk stats: %s", err)
		}
		sectorsWritten, err := strconv.Atoi(fieldRow[9])
		if err != nil {
			log.Printf("could not parse disk stats: %s", err)
		}
		manager.UpdateDiskIo(fieldRow[2], "read", float64(sectorsRead*sectorSize/MB))
		manager.UpdateDiskIo(fieldRow[2], "write", float64(sectorsWritten*sectorSize/MB))
	}
	return nil
}

func RefreshNetworkCounters(manager MetricsManager) error {
	statsFile, err := os.Open("/proc/net/dev")
	if err != nil {
		return fmt.Errorf("could not open /proc/diskstats: %s", err)
	}
	defer statsFile.Close()

	omitLinesCount := 0
	scanner := bufio.NewScanner(statsFile)
	for scanner.Scan() {
		if omitLinesCount < 2 {
			omitLinesCount++
			continue
		}
		line := scanner.Text()
		line = strings.TrimSpace(line)
		fieldRow := strings.Fields(line)

		bytesReceived, err := strconv.Atoi(fieldRow[1])
		if err != nil {
			log.Printf("could not parse disk stats: %s", err)
		}
		bytesSent, err := strconv.Atoi(fieldRow[9])
		if err != nil {
			log.Printf("could not parse disk stats: %s", err)
		}

		manager.UpdateNetworkIo(fieldRow[0], "received", float64(bytesReceived/MB))
		manager.UpdateNetworkIo(fieldRow[0], "sent", float64(bytesSent/MB))
	}
	return nil
}

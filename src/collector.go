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

type Metrics struct {
	CPUUsage        int
	MemoryUsage     int
	NetworkCounters map[string]NetworkStats
	DiskCounters    map[string]DiskStats
}

func (m *Metrics) Collect() {
	err := m.RefreshCpuUsage()
	if err != nil {
		log.Printf("Error collecting cpu usage: %v", err)
	}

	err = m.RefreshMemoryUsage()
	if err != nil {
		log.Printf("Error collecting memory usage: %v", err)
	}

	err = m.RefreshNetworkCounters()
	if err != nil {
		log.Printf("Error collecting network counters: %v", err)
	}

	err = m.RefreshDiskCounters()
	if err != nil {
		log.Printf("Error collecting disk counters: %v", err)
	}
}
func (m *Metrics) RefreshMemoryUsage() error {
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
	m.MemoryUsage = result["MemUse"] * 100 / (freeMem + result["MemUse"])
	return nil
}

var prevCPUReading [10]int

func (m *Metrics) RefreshCpuUsage() error {
	statsFile, err := os.Open("/proc/stat")
	if err != nil {
		return fmt.Errorf("could not open /proc/diskstats: %s", err)
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
			return fmt.Errorf("could not convert total value for field %s", field)
		}
		if k == 4 {
			idle = fieldValue - prevCPUReading[k-1]
		}
		total = total + fieldValue - prevCPUReading[k-1]
		prevCPUReading[k-1] = fieldValue
	}

	if total > 0 {
		m.CPUUsage = 100 - idle*100/total
	}
	return nil
}

type DiskStats struct {
	ReadBytes  uint64
	WriteBytes uint64
}

const sectorSize = 512

func (m *Metrics) RefreshDiskCounters() error {
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
		var diskStats DiskStats
		sectorsRead, err := strconv.Atoi(fieldRow[5])
		if err != nil {
			log.Printf("could not parse disk stats: %s", err)
		}
		sectorsWritten, err := strconv.Atoi(fieldRow[9])
		if err != nil {
			log.Printf("could not parse disk stats: %s", err)
		}
		diskStats.ReadBytes = uint64(sectorsRead * sectorSize)
		diskStats.WriteBytes = uint64(sectorsWritten * sectorSize)
		m.DiskCounters[fieldRow[2]] = diskStats
	}
	return nil
}

type NetworkStats struct {
	BytesSent int
	BytesRecv int
}

func (m *Metrics) RefreshNetworkCounters() error {
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
		var networkStats NetworkStats
		networkStats.BytesRecv, err = strconv.Atoi(fieldRow[1])
		if err != nil {
			log.Printf("could not parse disk stats: %s", err)
		}
		networkStats.BytesSent, err = strconv.Atoi(fieldRow[9])
		if err != nil {
			log.Printf("could not parse disk stats: %s", err)
		}

		m.NetworkCounters[fieldRow[0]] = networkStats
	}
	return nil
}

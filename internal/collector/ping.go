package collector

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var rttPattern = regexp.MustCompile(`(?i)time[=<]\s*([0-9]*\.?[0-9]+)\s*ms`)

type PingResult struct {
	Online  bool
	Latency float64
	Error   string
}

func Ping(ctx context.Context, address string, timeout time.Duration) PingResult {
	args := buildPingArgs(address, timeout)
	cmd := exec.CommandContext(ctx, "ping", args...)
	output, err := cmd.CombinedOutput()
	outStr := string(output)
	latency := parseLatency(outStr)

	result := PingResult{Latency: latency}
	if err != nil {
		result.Error = strings.TrimSpace(outStr)
		if result.Error == "" {
			result.Error = err.Error()
		}
		return result
	}

	result.Online = true
	return result
}

func buildPingArgs(address string, timeout time.Duration) []string {
	ms := int(timeout / time.Millisecond)
	if ms < 1 {
		ms = 1000
	}

	if runtime.GOOS == "windows" {
		return []string{"-n", "1", "-w", strconv.Itoa(ms), address}
	}

	sec := int(timeout / time.Second)
	if sec < 1 {
		sec = 1
	}
	return []string{"-c", "1", "-W", strconv.Itoa(sec), address}
}

func parseLatency(output string) float64 {
	match := rttPattern.FindStringSubmatch(output)
	if len(match) != 2 {
		if strings.Contains(strings.ToLower(output), "time<1ms") {
			return 1
		}
		return 0
	}

	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		fmt.Println("failed to parse latency:", err)
		return 0
	}
	return value
}

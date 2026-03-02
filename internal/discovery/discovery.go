package discovery

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"netscope/internal/collector"
	"netscope/internal/config"
)

var (
	nmapLinePattern = regexp.MustCompile(`(?i)^nmap scan report for (.+)$`)
	ipv4Pattern     = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
)

type Method string

const (
	MethodNmap Method = "nmap"
	MethodPing Method = "ping"
	MethodARP  Method = "arp"
	MethodAuto Method = "auto"
)

func Discover(ctx context.Context, method Method, subnet string, timeout time.Duration, workers int) ([]config.Device, error) {
	switch method {
	case MethodNmap:
		return DiscoverWithNmap(ctx, subnet)
	case MethodPing:
		return DiscoverWithPingSweep(ctx, subnet, timeout, workers)
	case MethodARP:
		return DiscoverWithARPTable(ctx, subnet)
	case MethodAuto:
		if _, err := exec.LookPath("nmap"); err == nil {
			return DiscoverWithNmap(ctx, subnet)
		}
		pingDevices, _ := DiscoverWithPingSweep(ctx, subnet, timeout, workers)
		arpDevices, _ := DiscoverWithARPTable(ctx, subnet)
		merged := mergeDevices(pingDevices, arpDevices)
		if len(merged) == 0 {
			return nil, fmt.Errorf("no hosts discovered via auto mode on %s", subnet)
		}
		return merged, nil
	default:
		return nil, fmt.Errorf("unsupported discovery method: %s", method)
	}
}

func DiscoverWithNmap(ctx context.Context, subnet string) ([]config.Device, error) {
	args := []string{"-sn", subnet}
	if runtime.GOOS == "windows" {
		args = append([]string{"-n"}, args...)
	}
	cmd := exec.CommandContext(ctx, "nmap", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run nmap: %w (%s)", err, strings.TrimSpace(string(output)))
	}

	devices := parseNmapHosts(string(output))
	if len(devices) == 0 {
		return nil, fmt.Errorf("no hosts discovered via nmap on %s", subnet)
	}
	return devices, nil
}

func DiscoverWithARPTable(ctx context.Context, subnet string) ([]config.Device, error) {
	cmd := exec.CommandContext(ctx, "arp", "-a")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run arp -a: %w", err)
	}
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet %q: %w", subnet, err)
	}

	seen := map[string]bool{}
	devices := make([]config.Device, 0)
	for _, ip := range ipv4Pattern.FindAllString(string(output), -1) {
		parsed := net.ParseIP(ip)
		if parsed == nil || !ipNet.Contains(parsed) || seen[ip] {
			continue
		}
		seen[ip] = true
		devices = append(devices, config.Device{Name: "host-" + strings.ReplaceAll(ip, ".", "-"), Address: ip})
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].Address < devices[j].Address })
	if len(devices) == 0 {
		return nil, fmt.Errorf("no hosts discovered via arp table on %s", subnet)
	}
	return devices, nil
}

func parseNmapHosts(output string) []config.Device {
	seen := map[string]bool{}
	devices := make([]config.Device, 0)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		match := nmapLinePattern.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}

		host, ip := extractHostAndIP(match[1])
		if ip == "" || seen[ip] {
			continue
		}
		if host == "" {
			host = "host-" + strings.ReplaceAll(ip, ".", "-")
		}
		seen[ip] = true
		devices = append(devices, config.Device{Name: host, Address: ip})
	}

	sort.Slice(devices, func(i, j int) bool { return devices[i].Address < devices[j].Address })
	return devices
}

func extractHostAndIP(value string) (string, string) {
	value = strings.TrimSpace(value)
	if strings.HasSuffix(value, ")") && strings.Contains(value, "(") {
		idx := strings.LastIndex(value, "(")
		host := strings.TrimSpace(value[:idx])
		ip := strings.TrimSuffix(value[idx+1:], ")")
		if net.ParseIP(ip) != nil {
			return host, ip
		}
	}
	if net.ParseIP(value) != nil {
		return "", value
	}
	return "", ""
}

func DiscoverWithPingSweep(ctx context.Context, subnet string, timeout time.Duration, workers int) ([]config.Device, error) {
	ips, err := hosts(subnet)
	if err != nil {
		return nil, err
	}
	if workers <= 0 {
		workers = 64
	}

	jobs := make(chan string)
	results := make(chan config.Device, len(ips))
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range jobs {
				res := collector.Ping(ctx, ip, timeout)
				if res.Online {
					name := "host-" + strings.ReplaceAll(ip, ".", "-")
					if host, err := net.LookupAddr(ip); err == nil && len(host) > 0 {
						name = strings.TrimSuffix(host[0], ".")
					}
					results <- config.Device{Name: name, Address: ip}
				}
			}
		}()
	}

	for _, ip := range ips {
		jobs <- ip
	}
	close(jobs)
	wg.Wait()
	close(results)

	devices := make([]config.Device, 0)
	for d := range results {
		devices = append(devices, d)
	}

	sort.Slice(devices, func(i, j int) bool { return devices[i].Address < devices[j].Address })
	if len(devices) == 0 {
		return nil, fmt.Errorf("no hosts discovered via ping sweep on %s", subnet)
	}
	return devices, nil
}

func mergeDevices(groups ...[]config.Device) []config.Device {
	seen := map[string]config.Device{}
	for _, g := range groups {
		for _, d := range g {
			if d.Address == "" {
				continue
			}
			if existing, ok := seen[d.Address]; ok {
				if strings.HasPrefix(existing.Name, "host-") && !strings.HasPrefix(d.Name, "host-") {
					seen[d.Address] = d
				}
				continue
			}
			seen[d.Address] = d
		}
	}
	devices := make([]config.Device, 0, len(seen))
	for _, d := range seen {
		devices = append(devices, d)
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].Address < devices[j].Address })
	return devices
}

func hosts(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet %q: %w", cidr, err)
	}
	ipv4 := ip.To4()
	if ipv4 == nil {
		return nil, fmt.Errorf("only IPv4 subnets supported")
	}

	var ips []string
	for ip := cloneIP(ipv4.Mask(ipnet.Mask)); ipnet.Contains(ip); incIP(ip) {
		ips = append(ips, ip.String())
	}
	if len(ips) <= 2 {
		return []string{}, nil
	}
	return ips[1 : len(ips)-1], nil
}

func cloneIP(ip net.IP) net.IP {
	out := make(net.IP, len(ip))
	copy(out, ip)
	return out
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

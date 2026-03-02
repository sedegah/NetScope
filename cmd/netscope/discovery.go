package main

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"sort"
	"strings"
	"sync"
	"time"

	"netscope/internal/collector"
	"netscope/internal/config"
)

type deviceProvider struct {
	mu      sync.RWMutex
	devices []config.Device
}

func newDeviceProvider(devices []config.Device) *deviceProvider {
	provider := &deviceProvider{}
	provider.Set(devices)
	return provider
}

func (p *deviceProvider) Set(devices []config.Device) {
	p.mu.Lock()
	defer p.mu.Unlock()
	copyDevices := make([]config.Device, len(devices))
	copy(copyDevices, devices)
	p.devices = copyDevices
}

func (p *deviceProvider) List() []config.Device {
	p.mu.RLock()
	defer p.mu.RUnlock()
	copyDevices := make([]config.Device, len(p.devices))
	copy(copyDevices, p.devices)
	return copyDevices
}

func startAutoDiscovery(ctx context.Context, provider *deviceProvider, subnet, method string, refresh time.Duration) error {
	if refresh <= 0 {
		return fmt.Errorf("-auto-refresh must be > 0")
	}

	devices, err := discoverDevices(ctx, subnet, method)
	if err != nil {
		return err
	}
	provider.Set(devices)
	log.Printf("auto-discovery: found %d reachable device(s) in %s\n", len(devices), subnet)

	go func() {
		ticker := time.NewTicker(refresh)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				discovered, discoverErr := discoverDevices(ctx, subnet, method)
				if discoverErr != nil {
					log.Printf("auto-discovery refresh failed: %v\n", discoverErr)
					continue
				}
				provider.Set(discovered)
				log.Printf("auto-discovery refresh: found %d reachable device(s)\n", len(discovered))
			}
		}
	}()

	return nil
}

func discoverDevices(ctx context.Context, subnet, method string) ([]config.Device, error) {
	if method != "auto" && method != "ping" {
		return nil, fmt.Errorf("unsupported -auto-method %q (supported: auto, ping)", method)
	}

	prefix, err := netip.ParsePrefix(subnet)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet %q: %w", subnet, err)
	}
	if !prefix.Addr().Is4() {
		return nil, fmt.Errorf("only IPv4 subnets are currently supported")
	}

	rangeStart, rangeEnd := hostRange(prefix)
	if rangeEnd < rangeStart {
		return nil, nil
	}

	const workerCount = 64
	type candidate struct {
		ip     string
	}

	jobs := make(chan candidate)
	results := make(chan config.Device, 32)
	var workers sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for job := range jobs {
				pingCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
				result := collector.Ping(pingCtx, job.ip, 1200*time.Millisecond)
				cancel()
				if result.Online {
					results <- config.Device{
						Name:    defaultName(job.ip),
						Address: job.ip,
					}
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for value := rangeStart; ; {
			select {
			case <-ctx.Done():
				return
			case jobs <- candidate{ip: uint32ToIP(value).String()}:
			}
			if value == rangeEnd {
				return
			}
			value++
		}
	}()

	go func() {
		workers.Wait()
		close(results)
	}()

	discovered := make([]config.Device, 0)
	for d := range results {
		discovered = append(discovered, d)
	}

	sort.Slice(discovered, func(i, j int) bool {
		return ipToUint32(discovered[i].Address) < ipToUint32(discovered[j].Address)
	})

	return discovered, nil
}

func hostRange(prefix netip.Prefix) (uint32, uint32) {
	network := ipToUint32(prefix.Masked().Addr().String())
	bits := prefix.Bits()
	if bits >= 31 {
		if bits == 32 {
			return network, network
		}
		return network, network + 1
	}

	hostCount := uint32(1) << uint32(32-bits)
	broadcast := network + hostCount - 1
	return network + 1, broadcast - 1
}

func defaultName(ip string) string {
	return "host-" + strings.ReplaceAll(ip, ".", "-")
}

func ipToUint32(ip string) uint32 {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return 0
	}
	v4 := addr.As4()
	return uint32(v4[0])<<24 | uint32(v4[1])<<16 | uint32(v4[2])<<8 | uint32(v4[3])
}

func uint32ToIP(value uint32) netip.Addr {
	return netip.AddrFrom4([4]byte{
		byte(value >> 24),
		byte(value >> 16),
		byte(value >> 8),
		byte(value),
	})
}

package monitor

import (
	"context"
	"time"

	"netscope/internal/collector"
	"netscope/internal/config"
	"netscope/internal/store"
)

type Service struct {
	Store *store.MemoryStore
}

func NewService(s *store.MemoryStore) *Service {
	return &Service{Store: s}
}

func (s *Service) ProbeDevice(ctx context.Context, device config.Device, probes int, timeout time.Duration) store.Snapshot {
	if probes <= 0 {
		probes = 4
	}

	successes := 0
	totalLatency := 0.0
	var lastErr string
	now := time.Now()
	lastSeen := time.Time{}

	for i := 0; i < probes; i++ {
		result := collector.Ping(ctx, device.Address, timeout)
		if result.Online {
			successes++
			totalLatency += result.Latency
			lastSeen = now
		} else {
			lastErr = result.Error
		}
	}

	loss := 100.0
	avgLatency := 0.0
	online := successes > 0
	if successes > 0 {
		loss = 100 - (float64(successes)/float64(probes))*100
		avgLatency = totalLatency / float64(successes)
	}

	snapshot := store.Snapshot{
		Name:       device.Name,
		Address:    device.Address,
		Online:     online,
		LatencyMS:  avgLatency,
		PacketLoss: loss,
		LastSeen:   lastSeen,
		UpdatedAt:  now,
		Error:      lastErr,
	}
	s.Store.Upsert(snapshot)
	return snapshot
}

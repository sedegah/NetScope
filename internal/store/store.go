package store

import (
	"sync"
	"time"
)

type Snapshot struct {
	Name       string    `json:"name"`
	Address    string    `json:"address"`
	Type       string    `json:"type,omitempty"`
	Online     bool      `json:"online"`
	LatencyMS  float64   `json:"latency_ms"`
	PacketLoss float64   `json:"packet_loss_percent"`
	LastSeen   time.Time `json:"last_seen"`
	UpdatedAt  time.Time `json:"updated_at"`
	Error      string    `json:"error,omitempty"`
}

type MemoryStore struct {
	mu      sync.RWMutex
	latest  map[string]Snapshot
	history map[string][]Snapshot
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		latest:  map[string]Snapshot{},
		history: map[string][]Snapshot{},
	}
}

func (s *MemoryStore) Upsert(snapshot Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.latest[snapshot.Address] = snapshot
	s.history[snapshot.Address] = append(s.history[snapshot.Address], snapshot)
}

func (s *MemoryStore) ListLatest() []Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Snapshot, 0, len(s.latest))
	for _, snapshot := range s.latest {
		result = append(result, snapshot)
	}
	return result
}

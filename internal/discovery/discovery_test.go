package discovery

import (
	"testing"

	"netscope/internal/config"
)

func TestParseNmapHosts(t *testing.T) {
	out := `Nmap scan report for router.lan (192.168.0.1)
Nmap scan report for 192.168.0.10
Nmap scan report for nas.local (192.168.0.50)
Nmap done: 256 IP addresses (3 hosts up) scanned`

	devices := parseNmapHosts(out)
	if len(devices) != 3 {
		t.Fatalf("expected 3 devices, got %d", len(devices))
	}
	if devices[0].Address != "192.168.0.1" {
		t.Fatalf("unexpected first device %+v", devices[0])
	}
}

func TestHosts(t *testing.T) {
	ips, err := hosts("192.168.1.0/30")
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(ips))
	}
	if ips[0] != "192.168.1.1" || ips[1] != "192.168.1.2" {
		t.Fatalf("unexpected hosts: %#v", ips)
	}
}

func TestMergeDevices(t *testing.T) {
	a := []config.Device{{Name: "host-192-168-0-10", Address: "192.168.0.10"}}
	b := []config.Device{{Name: "laptop.local", Address: "192.168.0.10"}, {Name: "router", Address: "192.168.0.1"}}
	merged := mergeDevices(a, b)
	if len(merged) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(merged))
	}
	if merged[1].Name != "laptop.local" {
		t.Fatalf("expected named host preference, got %+v", merged[1])
	}
}

# NetScope

NetScope is a lightweight, read-only network visibility and health monitoring tool for learners and entry-level network engineers.

## Features
- Device monitoring from a manual list (`devices.json`)
- ICMP availability (up/down)
- Latency and packet loss metrics
- CLI monitoring mode
- Optional web dashboard (`/api/status` + simple UI)
- Cross-platform operation (Linux and Windows)

## Project Architecture
- **Collector layer:** ICMP probing
- **Processing layer:** packet loss/latency aggregation
- **Storage layer:** in-memory latest + history snapshots
- **Presentation layer:** CLI and web dashboard

## Prerequisites
- Go 1.22+

## Quick start
```bash
cp devices.example.json devices.json
go run ./cmd/netscope monitor -config devices.json -interval 5s
```

Run the dashboard:
```bash
go run ./cmd/netscope web -config devices.json -listen :8080
```
Then open http://localhost:8080

## Build for Linux and Windows
### On Linux/macOS shell
```bash
./build.sh
```

### On PowerShell (Windows)
```powershell
./build.ps1
```

Both scripts generate:
- `dist/netscope-linux-amd64`
- `dist/netscope-windows-amd64.exe`

## Config format
`devices.json`
```json
{
  "devices": [
    {"name": "Gateway", "address": "192.168.1.1"},
    {"name": "DNS", "address": "8.8.8.8"}
  ]
}
```

## Notes
- NetScope only performs read-only checks.
- ICMP may require firewall permissions.
- ARP/SNMP collectors are intentionally left as future extensible modules for MVP simplicity.

# NetScope

NetScope is a lightweight, read-only network visibility and health monitoring tool for learners and entry-level network engineers.

## Features
- Real LAN/Wi-Fi device discovery (Nmap, ping sweep, ARP table)
- Fully automatic mode: discovery → `devices.json` update → monitoring loop
- ICMP availability (UP/DOWN)
- Latency and packet loss metrics
- CLI monitoring mode
- Optional web dashboard (`/api/status` + simple UI)
- Cross-platform operation and builds (Linux and Windows)

## Prerequisites
- Go 1.22+
- Optional (recommended for discovery): `nmap`

## Manual discovery (one-shot)

```bash
go run ./cmd/netscope discover -subnet 192.168.0.0/24 -method auto -output devices.json
```

Discovery methods:
- `auto`: use Nmap if available, otherwise combine ping sweep + ARP table
- `nmap`: force Nmap scan (`nmap -sn`)
- `ping`: force ping sweep
- `arp`: use local ARP table (`arp -a`)

## Fully automatic monitoring (real devices)

NetScope can continuously rediscover devices and update `devices.json` without manual edits.

### Automatic CLI monitoring
```bash
go run ./cmd/netscope monitor \
  -auto-subnet 192.168.0.0/24 \
  -auto-method auto \
  -auto-refresh 30s \
  -interval 5s \
  -config devices.json
```

### Automatic web monitoring
```bash
go run ./cmd/netscope web \
  -auto-subnet 192.168.0.0/24 \
  -auto-method auto \
  -auto-refresh 30s \
  -interval 5s \
  -config devices.json \
  -listen :8080
```

Open: http://localhost:8080

## devices.json format

```json
{
  "devices": [
    {"name": "Router", "address": "192.168.0.1", "type": "router"},
    {"name": "Laptop", "address": "192.168.0.10", "type": "pc"},
    {"name": "NAS", "address": "192.168.0.50", "type": "nas"}
  ]
}
```

Fields:
- `name`: friendly identifier
- `address`: IPv4 address
- `type` (optional): router/switch/pc/nas/etc.


## Windows run instructions (PowerShell)

From the project root (`NetScope`):

```powershell
copy .\devices.example.json .\devices.json
go run .\cmd\netscope monitor -config devices.json -interval 5s
```

For web mode:

```powershell
go run .\cmd\netscope web -config devices.json -listen :8080
```

### Troubleshooting: `syntax error` in `cmd\netscope\main.go`
If you see parser errors around specific line numbers:

1. Make sure you are on the latest commit of this branch.
2. Run a clean build from repo root:
   ```powershell
   go clean -cache
   go build .\cmd\netscope
   ```
3. Verify module root is detected correctly:
   ```powershell
   go env GOMOD
   ```
   It should point to this repo's `go.mod`.

## Build for Linux and Windows

### Linux/macOS shell
```bash
./build.sh
```

### PowerShell (Windows)
```powershell
./build.ps1
```

Build outputs:
- `dist/netscope-linux-amd64`
- `dist/netscope-windows-amd64.exe`

## Safety notes
- NetScope is read-only: no network configuration changes are made.
- ICMP reachability may be affected by host firewalls and ACLs.

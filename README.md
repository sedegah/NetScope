# NetScope

NetScope is a lightweight, read-only network visibility and health monitoring tool for learners and entry-level network engineers.

## Features
- Automatic subnet discovery and device monitoring
- ICMP availability (up/down)
- Latency and packet loss metrics
- CLI monitoring mode
- Optional web dashboard (`/api/status` + simple UI)
- Cross-platform operation (Linux and Windows)

## Prerequisites
- Go 1.22+

## Linux/macOS setup

### 1) Run monitor
```bash
go run ./cmd/netscope monitor \
  -auto-subnet 192.168.0.0/24 \
  -auto-method auto \
  -auto-refresh 30s \
  -interval 5s
```

### 2) Run web dashboard
```bash
go run ./cmd/netscope web -auto-subnet 192.168.0.0/24 -auto-refresh 30s -listen :8080
```

### 3) Build binaries
```bash
./build.sh
```

## Windows setup

### 1) Run monitor
PowerShell (one line):
```powershell
go run .\cmd\netscope monitor -auto-subnet 192.168.0.0/24 -auto-method auto -auto-refresh 30s -interval 5s
```

PowerShell (multiline):
```powershell
go run .\cmd\netscope monitor `
  -auto-subnet 192.168.0.0/24 `
  -auto-method auto `
  -auto-refresh 30s `
  -interval 5s
```

CMD:
```bat
go run .\cmd\netscope monitor -auto-subnet 192.168.0.0/24 -auto-method auto -auto-refresh 30s -interval 5s
```

### 2) Run web dashboard
PowerShell:
```powershell
go run .\cmd\netscope web -auto-subnet 192.168.0.0/24 -auto-refresh 30s -listen :8080
```

### 3) Build binaries
PowerShell:
```powershell
./build.ps1
```

Build outputs:
- `dist/netscope-linux-amd64`
- `dist/netscope-windows-amd64.exe`

### 4) Check available flags
```bash
go run ./cmd/netscope monitor -h
```

Open http://localhost:8080 for web mode.

## Notes
- NetScope only performs read-only checks.
- ICMP may require firewall permissions.

## Troubleshooting

### `-auto-refresh` / `-interval` "not recognized" in PowerShell
Cause: PowerShell does not use `\` for line continuation.

Fix:
- Use one line, or
- Use PowerShell backtick `` ` `` at line end for multiline.

### `syntax error` in `cmd\netscope\main.go` (line 159/174)
Cause: local file differs from current repository code.

Fix from repo root:
```bash
git pull
go clean -cache
go build ./...
```

If your local file was manually edited and broken, restore it:
```bash
git checkout -- cmd/netscope/main.go cmd/netscope/discovery.go
go build ./...
```

### `&&` is not valid in older PowerShell
Use either separate lines or `;` to chain commands.

$ErrorActionPreference = "Stop"
New-Item -ItemType Directory -Force -Path dist | Out-Null

Write-Host "Building Linux binary..."
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o dist/netscope-linux-amd64 ./cmd/netscope

Write-Host "Building Windows binary..."
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -o dist/netscope-windows-amd64.exe ./cmd/netscope

Write-Host "Done. Binaries in dist/"

$ErrorActionPreference = "Stop"
New-Item -ItemType Directory -Force -Path dist | Out-Null

function Assert-LastExitCode {
	param(
		[string]$Step
	)
	if ($LASTEXITCODE -ne 0) {
		throw "$Step failed with exit code $LASTEXITCODE"
	}
}

Write-Host "Building Linux binary..."
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o dist/netscope-linux-amd64 ./cmd/netscope
Assert-LastExitCode "Linux build"

Write-Host "Building Windows binary..."
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -o dist/netscope-windows-amd64.exe ./cmd/netscope
Assert-LastExitCode "Windows build"

Write-Host "Done. Binaries in dist/"

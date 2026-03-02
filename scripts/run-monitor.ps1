param(
  [string]$Config = "devices.json",
  [string]$Interval = "5s",
  [string]$AutoSubnet = "",
  [string]$AutoMethod = "auto",
  [string]$AutoRefresh = "30s"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $Config)) {
  Copy-Item .\devices.example.json $Config
}

$args = @("run", ".\cmd\netscope", "monitor", "-config", $Config, "-interval", $Interval)
if ($AutoSubnet -ne "") {
  $args += @("-auto-subnet", $AutoSubnet, "-auto-method", $AutoMethod, "-auto-refresh", $AutoRefresh)
}

go @args

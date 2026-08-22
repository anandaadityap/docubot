# Measure chat latency against a running DocuBot API.
# Usage: .\scripts\bench.ps1 [-BaseUrl http://127.0.0.1:8080]
param(
  [string]$BaseUrl = "http://127.0.0.1:8080",
  [string]$Message = "Gimana cara reset password?",
  [double]$OutPerMillion = 1.10
)

$health = Invoke-RestMethod -Uri "$BaseUrl/healthz"
Write-Host "== health ==" $health

$sw = [System.Diagnostics.Stopwatch]::StartNew()
$res = Invoke-WebRequest -Uri "$BaseUrl/api/v1/chat" -Method POST -ContentType "application/json" -Body (@{ message = $Message } | ConvertTo-Json) -TimeoutSec 90
$sw.Stop()

$body = $res.Content
$lat = 0
$tok = 0
if ($body -match '"latency_ms":(\d+)') { $lat = [int]$Matches[1] }
if ($body -match '"total_tokens":(\d+)') { $tok = [int]$Matches[1] }
$cost = [math]::Round(($tok / 1000000.0) * $OutPerMillion, 6)

Write-Host "== chat =="
Write-Host "total_wall_ms=$($sw.ElapsedMilliseconds)"
Write-Host "server_latency_ms=$lat"
Write-Host "total_tokens=$tok"
Write-Host "est_cost_usd=$cost  (using $OutPerMillion USD / 1M output tokens)"
Write-Host ""
Write-Host "== first SSE bytes =="
Write-Host ($body.Substring(0, [Math]::Min(400, $body.Length)))

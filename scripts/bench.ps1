# Measure chat latency against a running DocuBot API, including time-to-first-token.
# Usage: .\scripts\bench.ps1 [-BaseUrl http://127.0.0.1:8080]
param(
  [string]$BaseUrl = "http://127.0.0.1:8080",
  [string]$Message = "Gimana cara reset password?",
  [string]$Slug = "",
  [double]$OutPerMillion = 1.10
)

$health = Invoke-RestMethod -Uri "$BaseUrl/healthz"
Write-Host "== health ==" $health

if ([string]::IsNullOrWhiteSpace($Slug)) {
  $demo = Invoke-RestMethod -Uri "$BaseUrl/api/v1/demo"
  $Slug = $demo.data.slug
}
if ([string]::IsNullOrWhiteSpace($Slug)) {
  throw "no slug: pass -Slug or register a bot first (GET /api/v1/demo)"
}

$sw = [System.Diagnostics.Stopwatch]::StartNew()
$req = [System.Net.HttpWebRequest]::Create("$BaseUrl/api/v1/b/$Slug/chat")
$req.Method = "POST"
$req.ContentType = "application/json"
$req.Timeout = 90000
$bytes = [Text.Encoding]::UTF8.GetBytes((@{ message = $Message } | ConvertTo-Json))
$req.ContentLength = $bytes.Length
$rs = $req.GetRequestStream()
$rs.Write($bytes, 0, $bytes.Length)
$rs.Close()
$resp = $req.GetResponse()
$reader = New-Object System.IO.StreamReader($resp.GetResponseStream())
$body = New-Object System.Text.StringBuilder
$ttft = $null
while (($line = $reader.ReadLine()) -ne $null) {
  [void]$body.AppendLine($line)
  if ($null -eq $ttft -and $line -eq "event: token") {
    $ttft = $sw.ElapsedMilliseconds
  }
}
$sw.Stop()
$reader.Close()
$resp.Close()

$text = $body.ToString()
$lat = 0
$tok = 0
if ($text -match '"latency_ms":(\d+)') { $lat = [int]$Matches[1] }
if ($text -match '"total_tokens":(\d+)') { $tok = [int]$Matches[1] }
$cost = [math]::Round(($tok / 1000000.0) * $OutPerMillion, 6)
$ttftOut = if ($null -eq $ttft) { "n/a" } else { "$ttft" }

Write-Host "== chat =="
Write-Host "total_wall_ms=$($sw.ElapsedMilliseconds)"
Write-Host "ttft_ms=$ttftOut"
Write-Host "server_latency_ms=$lat"
Write-Host "total_tokens=$tok"
Write-Host "est_cost_usd=$cost  (using $OutPerMillion USD / 1M output tokens)"
Write-Host ""
Write-Host "== first SSE bytes =="
Write-Host ($text.Substring(0, [Math]::Min(400, $text.Length)))

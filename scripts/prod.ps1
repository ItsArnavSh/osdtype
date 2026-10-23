# Production stack: adapter-node frontend + compiled Go API.
# Dev login is explicitly disabled in this path.
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$webPort = if ($env:WEB_PORT) { $env:WEB_PORT } else { "3000" }
$apiPort = if ($env:API_PORT) { $env:API_PORT } else { "8080" }

$secretFile = Join-Path $root ".prod-secret"
if (-not (Test-Path $secretFile)) {
	$bytes = New-Object byte[] 32
	$rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
	$rng.GetBytes($bytes)
	$hex = ($bytes | ForEach-Object { $_.ToString("x2") }) -join ""
	Set-Content -Path $secretFile -Value $hex -NoNewline
}
$jwt = if ($env:JWTKEY) { $env:JWTKEY } else { (Get-Content $secretFile -Raw).Trim() }

Write-Host "==> building frontend"
Push-Location (Join-Path $root "app")
& npm.cmd install
& npm.cmd run build
Pop-Location

Write-Host "==> building api"
Push-Location (Join-Path $root "server")
& go build -trimpath -o bin/osdtyp.exe .
Pop-Location

Write-Host "==> api      http://localhost:$apiPort"
Write-Host "==> web      http://localhost:$webPort"
Write-Host "==> dev login disabled"

$env:PORT = $webPort
$env:ORIGIN = "http://localhost:$webPort"
$web = Start-Process -FilePath "node" -ArgumentList "build" `
	-WorkingDirectory (Join-Path $root "app") -NoNewWindow -PassThru

$env:PORT = ":$apiPort"
$env:DEV_LOGIN = "false"
$env:FRONTEND_URL = "http://localhost:$webPort"
$env:COOKIE_SECURE = "false"
$env:JWTKEY = $jwt
$server = Start-Process -FilePath (Join-Path $root "server\bin\osdtyp.exe") `
	-WorkingDirectory (Join-Path $root "server") -NoNewWindow -PassThru

try {
	Wait-Process -Id $server.Id, $web.Id
} finally {
	if (-not $server.HasExited) { Stop-Process -Id $server.Id -Force -ErrorAction SilentlyContinue }
	if (-not $web.HasExited) { Stop-Process -Id $web.Id -Force -ErrorAction SilentlyContinue }
}

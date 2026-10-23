# Development stack: Go API + Vite dev server.
# Prerequisites: Go 1.26+, Node 22+, PostgreSQL on 5432, Redis on 6379.
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$server = $null
$app = $null

function Stop-Stack {
	if ($server -and -not $server.HasExited) { Stop-Process -Id $server.Id -Force -ErrorAction SilentlyContinue }
	if ($app -and -not $app.HasExited) { Stop-Process -Id $app.Id -Force -ErrorAction SilentlyContinue }
}

try {
	if (-not (Test-Path (Join-Path $root "app\node_modules"))) {
		Write-Host "==> installing frontend dependencies"
		Push-Location (Join-Path $root "app")
		& npm.cmd install
		Pop-Location
	}

	Write-Host "==> api      http://localhost:8080  (dev login enabled)"
	Write-Host "==> web      http://localhost:5173"
	Write-Host "==> press ctrl+c to stop both"

	$env:PORT = ":8080"
	$env:DEV_LOGIN = "true"
	$env:FRONTEND_URL = "http://localhost:5173"
	$env:COOKIE_SECURE = "false"
	$server = Start-Process -FilePath "go" -ArgumentList "run", "." `
		-WorkingDirectory (Join-Path $root "server") -NoNewWindow -PassThru

	$app = Start-Process -FilePath "cmd.exe" -ArgumentList "/c", "npm.cmd", "run", "dev", "--", "--host", "localhost" `
		-WorkingDirectory (Join-Path $root "app") -NoNewWindow -PassThru

	Wait-Process -Id $server.Id, $app.Id
} finally {
	Stop-Stack
}

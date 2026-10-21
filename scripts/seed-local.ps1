$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

Import-Module (Join-Path $PSScriptRoot "local-common.psm1") -Force -DisableNameChecking
$Root = Split-Path -Parent $PSScriptRoot

Start-Postgres
Wait-Postgres

$seedPath = Join-Path $PSScriptRoot "seed-local.sql"
Write-Step "Sembrando datos locales de desarrollo"
$localPsql = Join-Path $Root ".local\postgres\pgsql\bin\psql.exe"
if (Test-Path $localPsql) {
  $env:PGPASSWORD = "postgres_password"
  Get-Content -Raw $seedPath | & $localPsql -h localhost -p 5432 -U postgres -d mesa_click -v ON_ERROR_STOP=1
} else {
  Require-Command docker
  Get-Content -Raw $seedPath | docker exec -i mesa_click_db psql -U postgres -d mesa_click -v ON_ERROR_STOP=1
}

if ($LASTEXITCODE -ne 0) {
  throw "El seed local falló. Revisá el error SQL mostrado arriba."
}

Write-Host "Seed local aplicado." -ForegroundColor Green
Write-Host "Usuario local: admin@mesaclick.local" -ForegroundColor Yellow
Write-Host "Clave local:   MesaClick2026" -ForegroundColor Yellow
$lanIP = Get-LocalLanIP
Write-Host "Mesa pública demo PC: http://localhost:3000/mesa/mesa-demo-1" -ForegroundColor Yellow
Write-Host "Mesa pública demo celular: http://${lanIP}:3000/mesa/mesa-demo-1" -ForegroundColor Yellow


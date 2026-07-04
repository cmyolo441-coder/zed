# ZED terminal AI agent — one-line installer (Windows)
#
# Usage (PowerShell me paste karo):
#   irm https://raw.githubusercontent.com/cmyolo441-coder/zed/main/install.ps1 | iex
#
# Ye script latest zed.exe GitHub Release se download karke
# %USERPROFILE%\.local\bin\zed.exe me install kar deti hai. Go ki zaroorat nahi.

$ErrorActionPreference = "Stop"

$repo = "cmyolo441-coder/zed"
$asset = "zed-windows-amd64.exe"
$installDir = Join-Path $env:USERPROFILE ".local\bin"
$target = Join-Path $installDir "zed.exe"
$url = "https://github.com/$repo/releases/latest/download/$asset"

Write-Host ""
Write-Host "  Installing ZED..." -ForegroundColor Cyan
Write-Host "  Source: $url" -ForegroundColor DarkGray
Write-Host ""

# --- install dir banao ---
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

# --- download ---
try {
    Invoke-WebRequest -Uri $url -OutFile $target -UseBasicParsing
} catch {
    Write-Host "  Download failed. Kya release me '$asset' asset hai?" -ForegroundColor Red
    exit 1
}

Write-Host "  Installed: $target" -ForegroundColor Green

# --- PATH me add karo (persistent) ---
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$installDir;$userPath", "User")
    $env:Path = "$installDir;$env:Path"
    Write-Host "  Added to PATH: $installDir" -ForegroundColor Green
    Write-Host "  (Naya terminal kholo taaki PATH update ho)" -ForegroundColor Yellow
} else {
    Write-Host "  Already in PATH" -ForegroundColor DarkGray
}

Write-Host ""
Write-Host "  Done! Naya terminal khol ke type karo:  zed" -ForegroundColor Cyan
Write-Host ""

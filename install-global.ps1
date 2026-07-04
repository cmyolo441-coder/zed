# install-global.ps1 — Make 'myagent' available system-wide on Windows.
# Run this ONCE:
#   cd C:\Users\hcmad\Desktop\zed1
#   .\install-global.ps1
#
# After that, open a NEW terminal and type:
#   myagent
#   myagent --model big-pickle
#   myagent /goal "build me a web scraper"

$ZedDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# --- Step 1: Create the myagent.bat shim in a persistent location ---
$shimDir = Join-Path $env:USERPROFILE ".local\bin"
if (-not (Test-Path $shimDir)) {
    New-Item -ItemType Directory -Path $shimDir -Force | Out-Null
    Write-Host "  Created: $shimDir" -ForegroundColor Green
}

$batPath = Join-Path $shimDir "myagent.bat"
$batContent = @"
@echo off
powershell -ExecutionPolicy Bypass -NoProfile -File "$ZedDir\myagent.ps1" %*
"@
Set-Content -Path $batPath -Value $batContent -Encoding ASCII
Write-Host "  Created: $batPath" -ForegroundColor Green

# --- Step 2: Add to user PATH (persistent, survives reboot) ---
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$shimDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$shimDir;$userPath", "User")
    $env:Path = "$shimDir;$env:Path"
    Write-Host "  Added $shimDir to user PATH" -ForegroundColor Green
} else {
    Write-Host "  $shimDir already in PATH" -ForegroundColor DarkGray
}

# --- Step 3: Verify config.json exists (key + model live there) ---
$cfgPath = Join-Path ([Environment]::GetFolderPath("ApplicationData")) "Zed\config.json"
if (Test-Path $cfgPath) {
    Write-Host "  Config OK: $cfgPath" -ForegroundColor Green
} else {
    Write-Host "  WARNING: $cfgPath not found." -ForegroundColor Yellow
    Write-Host "  The app will create a default config on first run." -ForegroundColor Yellow
}

# --- Done ---
Write-Host ""
Write-Host "  ==============================================" -ForegroundColor Cyan
Write-Host "   ZED Agent installed successfully!" -ForegroundColor Green
Write-Host "   " -ForegroundColor Cyan
Write-Host "   Close this terminal and open a NEW one." -ForegroundColor White
Write-Host "   Then just type:  myagent" -ForegroundColor White
Write-Host "   ==============================================" -ForegroundColor Cyan
Write-Host ""

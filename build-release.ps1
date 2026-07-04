# build-release.ps1 — Saare OS ke liye zed binaries banata hai.
# Chalane ke liye:  .\build-release.ps1
# Binaries 'dist' folder me aayenge. Inhe GitHub Release me upload karo.

$ErrorActionPreference = "Stop"
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $here

$dist = Join-Path $here "dist"
New-Item -ItemType Directory -Path $dist -Force | Out-Null

# Version tag (chaho to change kar sakte ho)
$version = "v1.0.0"

# Har target: GOOS, GOARCH, output filename
$targets = @(
    @{ os = "windows"; arch = "amd64"; out = "zed-windows-amd64.exe" },
    @{ os = "linux";   arch = "amd64"; out = "zed-linux-amd64" },
    @{ os = "linux";   arch = "arm64"; out = "zed-linux-arm64" },
    @{ os = "darwin";  arch = "amd64"; out = "zed-macos-amd64" },
    @{ os = "darwin";  arch = "arm64"; out = "zed-macos-arm64" }
)

Write-Host ""
Write-Host "  Building ZED $version for all platforms..." -ForegroundColor Cyan
Write-Host ""

foreach ($t in $targets) {
    $env:GOOS = $t.os
    $env:GOARCH = $t.arch
    $env:CGO_ENABLED = "0"
    $outPath = Join-Path $dist $t.out

    Write-Host "  -> $($t.os)/$($t.arch)  ..." -NoNewline
    go build -ldflags "-s -w" -o $outPath ./cmd/zed
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  OK  ($($t.out))" -ForegroundColor Green
    } else {
        Write-Host "  FAILED" -ForegroundColor Red
    }
}

# Environment reset
Remove-Item Env:\GOOS, Env:\GOARCH, Env:\CGO_ENABLED -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "  ==============================================" -ForegroundColor Cyan
Write-Host "   Done! Binaries yahan hain:" -ForegroundColor Green
Write-Host "   $dist" -ForegroundColor White
Write-Host "   Inhe GitHub Release me upload karo." -ForegroundColor White
Write-Host "  ==============================================" -ForegroundColor Cyan
Write-Host ""

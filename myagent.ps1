# myagent.ps1 — Launch ZED terminal AI agent from anywhere.
# Usage:  myagent
#         myagent --model z-ai/glm-5.2
#         myagent /goal "build a web scraper"
#
# Config lives in: %APPDATA%\Zed\config.json
# Currently set to: z-ai/glm-5.2 (1M context) via NVIDIA endpoint.
# The API key is baked into config.json, so no env var is required.

$ZedDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# Build if needed.
$exe = Join-Path $ZedDir "zed.exe"
if (-not (Test-Path $exe)) {
    Write-Host "  Building zed.exe ..." -ForegroundColor Cyan
    Push-Location $ZedDir
    go build -o zed.exe ./cmd/zed
    Pop-Location
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

# Pass arguments to zed.exe (e.g. --model, /goal, etc.)
& $exe @args

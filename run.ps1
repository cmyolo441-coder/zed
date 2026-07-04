# ZED launcher for Windows PowerShell.
# Usage:
#   1) Set your API key once (do NOT commit it):
#        $env:ZED_API_KEY = "your-key-here"
#   2) Run:
#        ./run.ps1                 # uses default model (mimo-v2.5-free)
#        ./run.ps1 big-pickle      # pick a model
#
# Available models: mimo-v2.5-free, deepseek-v4-flash-free, big-pickle

param(
    [string]$Model = "mimo-v2.5-free"
)

# OpenAI-compatible endpoint — only set defaults if config.json doesn't exist yet.
$cfgPath = Join-Path ([Environment]::GetFolderPath("ApplicationData")) "Zed\config.json"
if (-not (Test-Path $cfgPath)) {
    $env:ZED_PROVIDER = "openai"
    $env:ZED_BASE_URL = "https://opencode.ai/zen/v1/chat/completions"
    $env:ZED_MODEL    = $Model
}

if (-not $env:ZED_API_KEY) {
    Write-Host "ZED_API_KEY is not set." -ForegroundColor Yellow
    Write-Host 'Run:  $env:ZED_API_KEY = "your-key-here"' -ForegroundColor Yellow
    exit 1
}

# Build if the binary is missing, then launch.
if (-not (Test-Path "./zed.exe")) {
    Write-Host "Building zed.exe ..." -ForegroundColor Cyan
    go build -o zed.exe ./cmd/zed
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

./zed.exe

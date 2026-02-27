# PicoClaw Windows Setup Wizard

[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  PicoClaw Windows Setup Wizard" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "This wizard will help you configure PicoClaw" -ForegroundColor Yellow
Write-Host "You need:" -ForegroundColor Yellow
Write-Host "  1. API Key (OpenAI compatible API)" -ForegroundColor White
Write-Host "  2. Feishu App ID and App Secret" -ForegroundColor White
Write-Host ""

# Step 0: Run onboard to create directories (if needed)
Write-Host ""
Write-Host "[Step 0/3] Initializing PicoClaw directories..." -ForegroundColor Yellow
Write-Host "Running 'picoclaw onboard' to create necessary directories..."
& ".\picoclaw.exe" onboard *>$null
Write-Host "Done." -ForegroundColor Gray

# Step 1: Configure LLM
Write-Host ""
Write-Host "[Step 1/3] Configure LLM (OpenAI compatible API)" -ForegroundColor Yellow
Write-Host ""

$apiKey = ""
while ([string]::IsNullOrWhiteSpace($apiKey)) {
    $apiKey = Read-Host "Enter API Key"
    if ([string]::IsNullOrWhiteSpace($apiKey)) {
        Write-Host "Error: API Key cannot be empty" -ForegroundColor Red
    }
}

$apiBase = Read-Host "Enter API Base URL (Press Enter for default: https://api.openai.com/v1)"
if ([string]::IsNullOrWhiteSpace($apiBase)) {
    $apiBase = "https://api.openai.com/v1"
    Write-Host "Using default: $apiBase" -ForegroundColor Gray
}

$model = Read-Host "Enter model name (Press Enter for default: gpt-4)"
if ([string]::IsNullOrWhiteSpace($model)) {
    $model = "gpt-4"
    Write-Host "Using default: $model" -ForegroundColor Gray
}

# Step 2: Configure Feishu
Write-Host ""
Write-Host "[Step 2/3] Configure Feishu" -ForegroundColor Yellow
Write-Host ""

$feishuAppId = ""
while ([string]::IsNullOrWhiteSpace($feishuAppId)) {
    $feishuAppId = Read-Host "Enter Feishu App ID"
    if ([string]::IsNullOrWhiteSpace($feishuAppId)) {
        Write-Host "Error: Feishu App ID cannot be empty" -ForegroundColor Red
    }
}

$feishuAppSecret = ""
while ([string]::IsNullOrWhiteSpace($feishuAppSecret)) {
    $feishuAppSecret = Read-Host "Enter Feishu App Secret"
    if ([string]::IsNullOrWhiteSpace($feishuAppSecret)) {
        Write-Host "Error: Feishu App Secret cannot be empty" -ForegroundColor Red
    }
}

# Generate or update config file
Write-Host ""
Write-Host "[Step 3/3] Saving configuration..." -ForegroundColor Yellow

$configDir = Join-Path $env:USERPROFILE ".picoclaw"
if (-not (Test-Path $configDir)) {
    New-Item -Path $configDir -ItemType Directory -Force | Out-Null
}

$configPath = Join-Path $configDir "config.json"

# Read existing config (from onboard) or create minimal one
if (Test-Path $configPath) {
    Write-Host "Reading existing configuration..." -ForegroundColor Gray
    $configJson = Get-Content $configPath -Raw | ConvertFrom-Json
} else {
    Write-Host "Creating new configuration..." -ForegroundColor Gray
    $configJson = @{
        agents = @{ defaults = @{} }
        model_list = @()
        channels = @{}
    }
}

# Update model_list - use model name as the identifier
$modelEntry = @{
    model_name = $model    # This must match agents.defaults.model
    model = $model
    api_key = $apiKey
    api_base = $apiBase
}

$configJson.model_list = @($modelEntry)

# Update agents.defaults.model (NOT model_name)
if (-not $configJson.agents) {
    $configJson | Add-Member -MemberType NoteProperty -Name "agents" -Value @{} -Force
}
if (-not $configJson.agents.defaults) {
    $configJson.agents | Add-Member -MemberType NoteProperty -Name "defaults" -Value @{} -Force
}
$configJson.agents.defaults.model = $model

# Update channels.feishu
if (-not $configJson.channels) {
    $configJson | Add-Member -MemberType NoteProperty -Name "channels" -Value @{} -Force
}
if (-not $configJson.channels.feishu) {
    $configJson.channels | Add-Member -MemberType NoteProperty -Name "feishu" -Value @{} -Force
}

$configJson.channels.feishu | Add-Member -MemberType NoteProperty -Name "enabled" -Value $true -Force
$configJson.channels.feishu | Add-Member -MemberType NoteProperty -Name "app_id" -Value $feishuAppId -Force
$configJson.channels.feishu | Add-Member -MemberType NoteProperty -Name "app_secret" -Value $feishuAppSecret -Force

# Save configuration
$jsonContent = $configJson | ConvertTo-Json -Depth 10
[System.IO.File]::WriteAllText($configPath, $jsonContent, [System.Text.UTF8Encoding]::new($false))

Write-Host "Configuration saved to: $configPath" -ForegroundColor Green

# Start service
Write-Host ""
Write-Host "Configuration complete! Starting PicoClaw Gateway..." -ForegroundColor Green
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Service started" -ForegroundColor Cyan
Write-Host "  You can now chat via Feishu!" -ForegroundColor Cyan
Write-Host "  Press Ctrl+C to stop" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

& ".\picoclaw.exe" gateway

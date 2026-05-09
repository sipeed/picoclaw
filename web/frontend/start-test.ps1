# Start the preview server
$serverJob = Start-Job -ScriptBlock {
    param($port, $dir)
    Set-Location $dir
    pnpm preview --port $port
} -ArgumentList 5175, "C:\Users\user\Desktop\LEARN\AI\picoclaw\web\frontend"

# Wait for server to be ready
Start-Sleep -Seconds 5

# Check if server is ready
$ready = $false
for ($i = 0; $i -lt 30; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:5175" -Method Head -TimeoutSec 2 -ErrorAction SilentlyContinue
        if ($response.StatusCode -eq 200) {
            $ready = $true
            break
        }
    } catch {
        Start-Sleep -Seconds 1
    }
}

if ($ready) {
    Write-Host "Server is ready on port 5175"
    # Run Playwright tests
    Set-Location "C:\Users\user\Desktop\LEARN\AI\picoclaw\web\frontend"
    pnpm exec playwright test
} else {
    Write-Host "Server failed to start"
}

# Stop the server
Stop-Job -Job $serverJob -ErrorAction SilentlyContinue
Remove-Job -Job $serverJob -Force -ErrorAction SilentlyContinue
@echo off

echo ========================================
echo   PicoClaw Windows Setup Wizard
echo ========================================
echo.

REM Stop any running gateway instances
echo Stopping existing gateway instances...
taskkill /F /IM picoclaw.exe >nul 2>&1
if exist "%USERPROFILE%\.picoclaw\workspace\gateway.lock" del /F /Q "%USERPROFILE%\.picoclaw\workspace\gateway.lock" >nul 2>&1
echo.

echo Starting setup wizard...
echo.

PowerShell -NoProfile -ExecutionPolicy Bypass -File "%~dp0setup-wizard.ps1"

echo.
echo Tip: Press Ctrl+C to stop the service
echo.
pause

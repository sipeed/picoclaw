@echo off
REM Build script for picoclaw on Windows
REM Compiles the picoclaw binary for Windows x64

setlocal enabledelayedexpansion

REM Build variables
set BINARY_NAME=picoclaw
set BUILD_DIR=build
set CMD_DIR=cmd/picoclaw
set MAIN_GO=%CMD_DIR%/main.go

REM Get git information
for /f "tokens=*" %%i in ('git describe --tags --always --dirty 2^>nul') do set VERSION=%%i
if not defined VERSION set VERSION=dev

for /f "tokens=*" %%i in ('git rev-parse --short=8 HEAD 2^>nul') do set GIT_COMMIT=%%i
if not defined GIT_COMMIT set GIT_COMMIT=dev

for /f "tokens=*" %%i in ('powershell -Command "Get-Date -Format 'o'"') do set BUILD_TIME=%%i

REM Get Go version
for /f "tokens=3" %%i in ('go version') do set GO_VERSION=%%i

REM Set linker flags
set LDFLAGS=-ldflags "-X main.version=%VERSION% -X main.gitCommit=%GIT_COMMIT% -X main.buildTime=%BUILD_TIME% -X main.goVersion=%GO_VERSION% -s -w"

REM Run generate
echo Running go generate...
go generate ./...
if errorlevel 1 (
    echo Generate failed!
    exit /b 1
)

REM Create build directory
if not exist %BUILD_DIR% mkdir %BUILD_DIR%

REM Build for Windows x64
echo Building %BINARY_NAME% for Windows x64...
set GOOS=windows
set GOARCH=amd64
go build -v -tags stdjson %LDFLAGS% -o %BUILD_DIR%\%BINARY_NAME%-windows-amd64.exe ./%CMD_DIR%

if errorlevel 1 (
    echo Build failed!
    exit /b 1
)

echo Build complete: %BUILD_DIR%\%BINARY_NAME%-windows-amd64.exe
echo.
echo Build output:
dir %BUILD_DIR%\%BINARY_NAME%-windows-amd64.exe

endlocal

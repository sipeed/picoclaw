@echo off
REM Build PicoClaw core and web launcher without using make.
REM Usage: scripts\build-without-make.bat

SETLOCAL ENABLEEXTENSIONS

SET "GO_TAGS=goolm,stdjson"

REM Resolve the repository root directory from the script location.
PUSHD "%~dp0.."
IF ERRORLEVEL 1 (
    echo ERROR: Failed to resolve repository root from "%~dp0..".
    EXIT /B 1
)
SET "REPO_ROOT=%CD%"
POPD

REM Ensure Go is available
where go >nul 2>&1
IF ERRORLEVEL 1 (
    echo ERROR: go is not installed or not on PATH.
    EXIT /B 1
)

REM Ensure pnpm is available
where pnpm >nul 2>&1
IF ERRORLEVEL 1 (
    echo ERROR: pnpm is not installed or not on PATH.
    EXIT /B 1
)

PUSHD "%REPO_ROOT%"
IF ERRORLEVEL 1 (
    echo ERROR: Failed to change directory to "%REPO_ROOT%".
    EXIT /B 1
)

IF NOT EXIST build (
    mkdir build
)

echo === Generating Go code ===
go generate ./...
IF ERRORLEVEL 1 (
    echo ERROR: go generate failed.
    POPD
    EXIT /B 1
)

echo.
echo === Building PicoClaw core binary ===
go build -tags "%GO_TAGS%" -o build\picoclaw.exe .\cmd\picoclaw
IF ERRORLEVEL 1 (
    echo ERROR: go build failed for core binary.
    POPD
    EXIT /B 1
)

echo.
echo === Building PicoClaw web frontend assets ===
PUSHD "%REPO_ROOT%\web\frontend"
IF ERRORLEVEL 1 (
    echo ERROR: Failed to change directory to web\frontend.
    POPD
    EXIT /B 1
)
CALL pnpm install --frozen-lockfile
IF ERRORLEVEL 1 (
    echo ERROR: pnpm install failed.
    POPD
    POPD
    EXIT /B 1
)
CALL pnpm build:backend
IF ERRORLEVEL 1 (
    echo ERROR: pnpm build:backend failed.
    POPD
    POPD
    EXIT /B 1
)
POPd

echo.
echo === Building PicoClaw web launcher ===
go build -tags "%GO_TAGS%" -o build\picoclaw-launcher.exe .\web\backend
IF ERRORLEVEL 1 (
    echo ERROR: go build failed for web launcher.
    POPD
    EXIT /B 1
)

echo.
echo === Build complete ===
echo Core binary: %REPO_ROOT%\build\picoclaw.exe
echo Web launcher: %REPO_ROOT%\build\picoclaw-launcher.exe

POPD
EXIT /B 0

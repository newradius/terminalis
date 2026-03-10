@echo off
setlocal

echo === Terminalis Windows Build Script ===
echo.

:: Check for Go
where go >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Go is not installed or not in PATH.
    echo Download it from https://go.dev/dl/
    exit /b 1
)
echo [1/4] Go found:
go version

:: Check for Node.js
where node >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Node.js is not installed or not in PATH.
    echo Download it from https://nodejs.org/
    exit /b 1
)
echo [2/4] Node.js found:
node --version

:: Check for Wails CLI
where wails >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [3/4] Wails CLI not found. Installing...
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
    if %ERRORLEVEL% neq 0 (
        echo [ERROR] Failed to install Wails CLI.
        exit /b 1
    )
) else (
    echo [3/4] Wails CLI found.
)

:: Build
echo [4/4] Building Terminalis for Windows...
cd /d "%~dp0"
wails build -platform windows/amd64
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Build failed.
    exit /b 1
)

echo.
echo === Build complete! ===
echo Executable: %~dp0build\bin\Terminalis.exe

endlocal

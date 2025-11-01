@echo off
echo ========================================
echo Starting Backend Server with WebSocket Fix
echo ========================================
echo.

cd /d %~dp0
echo Working directory: %cd%
echo.

if exist "build\app.exe" (
    echo Starting server...
    echo Press Ctrl+C to stop the server
    echo ========================================
    echo.
    build\app.exe
) else (
    echo ERROR: build\app.exe not found!
    echo Please run: go build -o build/app.exe .
    pause
)


@echo off
setlocal

cd /d "%~dp0" || exit /b 1

set "CGO_ENABLED=0"
set "GOARCH=amd64"

set "GOOS=windows"
echo Building goi for %GOOS%/%GOARCH%...
go build -trimpath -ldflags="-s -w" -o goi.exe .
if errorlevel 1 (
    echo Windows build failed.
    exit /b 1
)

set "GOOS=linux"
echo Building goi for %GOOS%/%GOARCH%...
go build -trimpath -ldflags="-s -w" -o goi .
if errorlevel 1 (
    echo Linux build failed.
    exit /b 1
)

echo Build completed:
echo   Windows: %CD%\goi.exe
echo   Linux:   %CD%\goi
endlocal

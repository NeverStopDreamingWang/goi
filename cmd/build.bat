@echo off
setlocal

cd /d "%~dp0" || exit /b 1

set "CGO_ENABLED=0"
set "GOARCH=amd64"

set "WINDOWS_OUTPUT=%CD%\dist\windows-amd64\goi.exe"
set "LINUX_OUTPUT=%CD%\dist\linux-amd64\goi"

if not exist "%CD%\dist\windows-amd64" mkdir "%CD%\dist\windows-amd64"
if not exist "%CD%\dist\linux-amd64" mkdir "%CD%\dist\linux-amd64"

set "GOOS=windows"
echo Building goi for %GOOS%/%GOARCH%...
go build -trimpath -ldflags="-s -w" -o "%WINDOWS_OUTPUT%" ./goi
if errorlevel 1 (
    echo Windows build failed.
    exit /b 1
)

set "GOOS=linux"
echo Building goi for %GOOS%/%GOARCH%...
go build -trimpath -ldflags="-s -w" -o "%LINUX_OUTPUT%" ./goi
if errorlevel 1 (
    echo Linux build failed.
    exit /b 1
)

echo Build completed:
echo   Windows: %WINDOWS_OUTPUT%
echo   Linux:   %LINUX_OUTPUT%
endlocal

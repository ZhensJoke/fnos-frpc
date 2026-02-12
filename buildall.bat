@echo off
chcp 65001 >nul
setlocal

echo ============================================
echo   FNOS FRP 管理器 - 全平台编译脚本
echo ============================================
echo.

set PROJECT_DIR=%~dp0
set OUTPUT_DIR=%PROJECT_DIR%dist
set APP_NAME=fnos-frpc-gui

:: 清理旧的输出
if exist "%OUTPUT_DIR%" rmdir /s /q "%OUTPUT_DIR%"
mkdir "%OUTPUT_DIR%"

set CGO_ENABLED=0
set BUILD_FLAGS=-ldflags="-s -w"

:: ---- 1/3: Linux amd64 ----
echo [1/3] 编译 Linux amd64 ...
set GOOS=linux
set GOARCH=amd64
go build %BUILD_FLAGS% -o "%OUTPUT_DIR%\%APP_NAME%-linux-amd64" .
if %ERRORLEVEL% neq 0 (
    echo [错误] Linux amd64 编译失败！
    pause
    exit /b 1
)
echo       Linux amd64 ✓

:: ---- 2/3: Linux arm64 ----
echo [2/3] 编译 Linux arm64 ...
set GOOS=linux
set GOARCH=arm64
go build %BUILD_FLAGS% -o "%OUTPUT_DIR%\%APP_NAME%-linux-arm64" .
if %ERRORLEVEL% neq 0 (
    echo [错误] Linux arm64 编译失败！
    pause
    exit /b 1
)
echo       Linux arm64 ✓

:: ---- 3/3: Windows amd64 ----
echo [3/3] 编译 Windows amd64 ...
set GOOS=windows
set GOARCH=amd64
go build %BUILD_FLAGS% -o "%OUTPUT_DIR%\%APP_NAME%-windows-amd64.exe" .
if %ERRORLEVEL% neq 0 (
    echo [错误] Windows amd64 编译失败！
    pause
    exit /b 1
)
echo       Windows amd64 ✓

:: ---- 完成 ----
echo.
echo ============================================
echo   全平台编译完成！输出文件位于:
echo   %OUTPUT_DIR%
echo.
echo   dist/
echo     ├── %APP_NAME%-linux-amd64       (Linux x86_64)
echo     ├── %APP_NAME%-linux-arm64       (Linux ARM64)
echo     └── %APP_NAME%-windows-amd64.exe (Windows x64)
echo.
echo   静态资源已内嵌到二进制文件中，无需额外文件。
echo ============================================

pause

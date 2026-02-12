@echo off
chcp 65001 >nul
setlocal

echo ============================================
echo   FNOS FRP 管理器 - 一键打包脚本
echo ============================================
echo.

set PROJECT_DIR=%~dp0
set OUTPUT_DIR=%PROJECT_DIR%dist
set IMAGE_NAME=fnos-frpc-gui
set IMAGE_TAG=latest

:: 清理旧的输出
if exist "%OUTPUT_DIR%" rmdir /s /q "%OUTPUT_DIR%"
mkdir "%OUTPUT_DIR%"

:: ---- Step 1: 交叉编译 Linux 二进制 ----
echo [1/4] 交叉编译 Linux amd64 二进制...
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o "%OUTPUT_DIR%\fnos-frpc-gui" .
if %ERRORLEVEL% neq 0 (
    echo [错误] Go 编译失败！
    pause
    exit /b 1
)
echo       编译成功 ✓

:: ---- Step 2: 准备 Docker 构建上下文 ----
echo [2/4] 准备 Docker 构建文件...

:: 生成精简 Dockerfile（静态资源已内嵌到二进制中）
(
echo FROM alpine:3.19
echo RUN apk add --no-cache ca-certificates tzdata
echo WORKDIR /app
echo COPY fnos-frpc-gui .
echo VOLUME /app/data
echo ENV WEB_PORT=7500
echo ENTRYPOINT ["./fnos-frpc-gui"]
) > "%OUTPUT_DIR%\Dockerfile"

:: 复制 docker-compose.yml
copy /y "%PROJECT_DIR%docker-compose.yml" "%OUTPUT_DIR%\docker-compose.yml" >nul
echo       准备完成 ✓

:: ---- Step 3: 构建 Docker 镜像 ----
echo [3/4] 构建 Docker 镜像 %IMAGE_NAME%:%IMAGE_TAG% ...
docker build -t %IMAGE_NAME%:%IMAGE_TAG% "%OUTPUT_DIR%"
if %ERRORLEVEL% neq 0 (
    echo [错误] Docker 构建失败！请确认 Docker Desktop 已启动。
    pause
    exit /b 1
)
echo       镜像构建成功 ✓

:: ---- Step 4: 导出镜像 ----
echo [4/4] 导出镜像为 tar 文件...
docker save %IMAGE_NAME%:%IMAGE_TAG% -o "%OUTPUT_DIR%\%IMAGE_NAME%.tar"
if %ERRORLEVEL% neq 0 (
    echo [错误] 镜像导出失败！
    pause
    exit /b 1
)
echo       导出成功 ✓

:: ---- 完成 ----
echo.
echo ============================================
echo   打包完成！输出文件位于:
echo   %OUTPUT_DIR%
echo.
echo   dist/
echo     ├── fnos-frpc-gui.tar    (Docker 镜像)
echo     └── docker-compose.yml   (部署配置)
echo.
echo   部署到 fnos 步骤:
echo   1. 将上述两个文件上传到 NAS 同一目录
echo   2. SSH 到 NAS 执行:
echo      docker load -i fnos-frpc-gui.tar
echo      docker-compose up -d
echo   3. 浏览器访问 http://NAS-IP:7500
echo ============================================

pause

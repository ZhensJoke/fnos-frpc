#!/bin/bash
# FNOS FRP 管理器 - 一键安装脚本
# 用法: curl -fsSL https://raw.githubusercontent.com/ZhensJoke/fnos-frpc/main/install.sh | bash

set -e

# ---- 配置 ----
REPO="ZhensJoke/fnos-frpc"
INSTALL_DIR="/opt/fnos-frpc"
SERVICE_NAME="fnos-frpc"
PORT="${WEB_PORT:-7500}"

# ---- 检测架构 ----
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)   ARCH_NAME="linux-amd64" ;;
    aarch64|arm64)   ARCH_NAME="linux-arm64" ;;
    *)
        echo "❌ 不支持的架构: $ARCH"
        echo "   仅支持 x86_64 (amd64) 和 aarch64 (arm64)"
        exit 1
        ;;
esac

echo "============================================"
echo "  FNOS FRP 管理器 - 一键安装"
echo "============================================"
echo ""
echo "  架构: $ARCH ($ARCH_NAME)"
echo "  安装目录: $INSTALL_DIR"
echo "  Web 端口: $PORT"
echo ""

# ---- 获取最新版本 ----
echo "[1/4] 获取最新版本..."
LATEST_TAG=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
if [ -z "$LATEST_TAG" ]; then
    echo "❌ 无法获取最新版本，请检查网络连接"
    exit 1
fi
echo "      最新版本: $LATEST_TAG"

# ---- 检测已有安装 ----
IS_UPGRADE=false
OLD_VERSION=""
if [ -f "$INSTALL_DIR/fnos-frpc-gui" ]; then
    IS_UPGRADE=true
    # 尝试获取旧版本号
    if systemctl is-active --quiet $SERVICE_NAME 2>/dev/null; then
        OLD_VERSION="(正在运行)"
    fi
    echo ""
    echo "  ⚡ 检测到已安装的版本，将保留数据并升级"
    echo "     数据目录 $INSTALL_DIR/data/ 不会被删除"
    echo ""

    # 停止旧服务
    echo "[2/4] 停止旧版本服务..."
    systemctl stop $SERVICE_NAME 2>/dev/null || true
    echo "      已停止 ✓"
else
    echo ""
fi

# ---- 下载二进制文件 ----
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_TAG/fnos-frpc-gui-$ARCH_NAME"
STEP=$( [ "$IS_UPGRADE" = true ] && echo "3/4" || echo "2/4" )
echo "[$STEP] 下载 fnos-frpc-gui-$ARCH_NAME ..."
mkdir -p "$INSTALL_DIR/data"
curl -fsSL "$DOWNLOAD_URL" -o "$INSTALL_DIR/fnos-frpc-gui"
chmod +x "$INSTALL_DIR/fnos-frpc-gui"
echo "      下载完成 ✓"

# ---- 创建 systemd 服务 ----
STEP=$( [ "$IS_UPGRADE" = true ] && echo "3/4" || echo "3/4" )
echo "[$STEP] 创建系统服务..."
cat > /etc/systemd/system/$SERVICE_NAME.service << EOF
[Unit]
Description=FNOS FRP GUI Manager
After=network.target

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
Environment=WEB_PORT=$PORT
Environment=DATA_DIR=$INSTALL_DIR/data
ExecStart=$INSTALL_DIR/fnos-frpc-gui
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable $SERVICE_NAME
echo "      服务创建完成 ✓"

# ---- 启动服务 ----
echo "[4/4] 启动服务..."
systemctl restart $SERVICE_NAME
sleep 2

if systemctl is-active --quiet $SERVICE_NAME; then
    echo "      服务启动成功 ✓"
    echo ""
    echo "============================================"
    if [ "$IS_UPGRADE" = true ]; then
        echo "  ✅ 升级完成！版本: $LATEST_TAG"
        echo "     数据已保留，无需重新配置"
    else
        echo "  ✅ 安装完成！版本: $LATEST_TAG"
    fi
    echo ""
    echo "  访问地址: http://$(hostname -I | awk '{print $1}'):$PORT"
    echo ""
    echo "  常用命令:"
    echo "    查看状态:  systemctl status $SERVICE_NAME"
    echo "    查看日志:  journalctl -u $SERVICE_NAME -f"
    echo "    停止服务:  systemctl stop $SERVICE_NAME"
    echo "    卸载:      systemctl stop $SERVICE_NAME && rm -rf $INSTALL_DIR /etc/systemd/system/$SERVICE_NAME.service"
    echo "============================================"
else
    echo "❌ 服务启动失败，请检查日志:"
    echo "   journalctl -u $SERVICE_NAME -n 20"
    exit 1
fi

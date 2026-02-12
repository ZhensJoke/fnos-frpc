#!/bin/bash
# FNOS FRP 管理器 - NAS 端部署脚本
# 将此脚本和 fnos-frpc-gui.tar、docker-compose.yml 放在同一目录下

set -e

echo "============================================"
echo "  FNOS FRP 管理器 - 部署脚本"
echo "============================================"
echo ""

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# 检查文件
if [ ! -f "fnos-frpc-gui.tar" ]; then
    echo "[错误] 未找到 fnos-frpc-gui.tar 文件！"
    exit 1
fi

if [ ! -f "docker-compose.yml" ]; then
    echo "[错误] 未找到 docker-compose.yml 文件！"
    exit 1
fi

# 导入镜像
echo "[1/3] 导入 Docker 镜像..."
docker load -i fnos-frpc-gui.tar
echo "      导入成功 ✓"

# 创建数据目录
echo "[2/3] 创建数据目录..."
mkdir -p ./data
echo "      创建成功 ✓"

# 启动容器
echo "[3/3] 启动容器..."
docker compose up -d 2>/dev/null || docker-compose up -d
echo "      启动成功 ✓"

echo ""
echo "============================================"
echo "  部署完成！"
echo ""

# 获取 NAS IP
IP=$(hostname -I 2>/dev/null | awk '{print $1}' || echo "NAS-IP")
PORT=$(grep WEB_PORT docker-compose.yml | grep -o '[0-9]*' | head -1)
PORT=${PORT:-7500}

echo "  访问地址: http://${IP}:${PORT}"
echo ""
echo "  常用命令:"
echo "    查看状态: docker compose ps"
echo "    查看日志: docker compose logs -f"
echo "    停止服务: docker compose down"
echo "    重新启动: docker compose up -d"
echo "============================================"

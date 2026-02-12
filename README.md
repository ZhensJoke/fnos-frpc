# FNOS FRP 管理器

基于 Docker 的 FRP 客户端图形化配置工具，专为飞牛NAS (fnOS) 设计。

## 功能特性

- 🖥️ Web 图形界面管理 frpc 配置
- 📡 支持多个 frpc 服务器同时管理
- 🔄 支持 TCP/UDP/HTTP/HTTPS 代理类型
- 📦 frpc 在线安装（GitHub）和离线上传安装
- 🔒 首次使用设置管理密码
- 🐳 支持 Docker 部署 / 直接运行
- 💻 支持 Linux amd64 / arm64 / Windows 多平台

---

## 快速安装

### 方式一：一键安装（推荐）

SSH 登录飞牛NAS 后，执行以下命令即可自动下载并启动：

```bash
curl -fsSL https://raw.githubusercontent.com/ZhensJoke/fnos-frpc/main/install.sh | bash
```

脚本会自动完成：检测架构 → 下载最新版 → 创建系统服务 → 启动。

> 💡 自定义端口：`WEB_PORT=8080 curl -fsSL https://raw.githubusercontent.com/ZhensJoke/fnos-frpc/main/install.sh | bash`

### 方式二：Docker 部署

**1. 拉取镜像并运行：**

```bash
mkdir -p /vol1/docker/fnos-frpc && cd /vol1/docker/fnos-frpc

# 下载 docker-compose.yml
curl -fsSL https://raw.githubusercontent.com/ZhensJoke/fnos-frpc/main/docker-compose.yml -o docker-compose.yml

# 启动
docker compose up -d
```

**2. 或使用 docker run：**

```bash
docker run -d \
  --name fnos-frpc \
  --network host \
  -v ./data:/app/data \
  -e WEB_PORT=7500 \
  -e TZ=Asia/Shanghai \
  --restart unless-stopped \
  fnos-frpc-gui:latest
```

### 方式三：手动下载二进制

从 [Releases](https://github.com/ZhensJoke/fnos-frpc/releases) 页面下载对应平台的文件：

| 文件 | 平台 |
|------|------|
| `fnos-frpc-gui-linux-amd64` | Linux x86_64（大多数 NAS） |
| `fnos-frpc-gui-linux-arm64` | Linux ARM64（树莓派等） |
| `fnos-frpc-gui-windows-amd64.exe` | Windows x64 |

下载后直接运行：

```bash
chmod +x fnos-frpc-gui-linux-amd64
DATA_DIR=./data WEB_PORT=7500 ./fnos-frpc-gui-linux-amd64
```

---

## 访问管理界面

安装完成后，浏览器打开：

```
http://NAS-IP:7500
```

---

## 使用说明

### 1. 设置管理密码

首次访问会提示设置管理密码（至少 6 位）。设置完成后自动登录。

### 2. 安装 frpc

登录后，点击右上角的 🌐 按钮，进入 **frpc 版本管理**：

- **在线安装**：点击「在线安装 / 更新」，自动从 GitHub 下载最新版 frpc
- **离线安装**：如果 NAS 无法访问 GitHub，可在 [frp releases](https://github.com/fatedier/frp/releases) 手动下载 `frp_*_linux_amd64.tar.gz`，然后拖拽到上传区域

### 3. 添加 frps 服务器

点击左侧栏的 **+** 按钮，填写你的 frps 服务器信息：

| 字段 | 说明 | 示例 |
|------|------|------|
| 名称 | 自定义名称 | `我的VPS` |
| 服务器地址 | frps 服务器 IP 或域名 | `frps.example.com` |
| 端口 | frps 监听端口 | `7000` |
| Token | 与 frps 一致的认证 Token | `your_token` |

### 4. 添加代理规则

选中服务器后，点击「添加规则」：

**TCP 代理示例（SSH 远程访问）：**

| 字段 | 值 |
|------|---|
| 名称 | `ssh` |
| 类型 | `TCP` |
| 本地 IP | `127.0.0.1` |
| 本地端口 | `22` |
| 远程端口 | `6022` |

**HTTP 代理示例（NAS Web 界面）：**

| 字段 | 值 |
|------|---|
| 名称 | `nas-web` |
| 类型 | `HTTP` |
| 本地 IP | `127.0.0.1` |
| 本地端口 | `5666` |
| 自定义域名 | `nas.yourdomain.com` |

### 5. 启动连接

在服务器详情页点击「启动」按钮，frpc 即开始运行。底部的日志区域会实时显示连接状态。

---

## 服务管理

一键安装方式下，使用 systemd 管理服务：

```bash
# 查看状态
systemctl status fnos-frpc

# 查看日志
journalctl -u fnos-frpc -f

# 重启
systemctl restart fnos-frpc

# 停止
systemctl stop fnos-frpc
```

## 自定义端口

**一键安装方式：** 编辑服务文件中的 `WEB_PORT`：
```bash
systemctl edit fnos-frpc
# 添加: Environment=WEB_PORT=8080
systemctl restart fnos-frpc
```

**Docker 方式：** 修改 `docker-compose.yml` 中的 `WEB_PORT` 环境变量，然后重启：
```bash
docker compose down && docker compose up -d
```

## 数据持久化

配置数据保存在 `data/` 目录中，包括：

| 文件/目录 | 内容 |
|----------|------|
| `auth.json` | 管理密码（bcrypt 哈希） |
| `servers.json` | 服务器和代理规则配置 |
| `frpc/` | frpc 二进制文件 |
| `conf/` | 自动生成的 frpc TOML 配置 |
| `logs/` | frpc 运行日志 |

> ⚠️ 备份 NAS 时建议一并备份 `data/` 目录。

## 更新升级

**一键安装方式：** 重新执行安装脚本即可：
```bash
curl -fsSL https://raw.githubusercontent.com/ZhensJoke/fnos-frpc/main/install.sh | bash
```

**Docker 方式：**
```bash
docker compose down
docker compose pull
docker compose up -d
```

## 卸载

**一键安装方式：**
```bash
systemctl stop fnos-frpc
systemctl disable fnos-frpc
rm -rf /opt/fnos-frpc /etc/systemd/system/fnos-frpc.service
systemctl daemon-reload
```

**Docker 方式：**
```bash
docker compose down
```

## 技术栈

- 后端：Go（零外部依赖，仅标准库，静态资源内嵌）
- 前端：HTML/CSS/JS（无框架）
- 支持平台：Linux amd64 / arm64、Windows amd64
- Docker 镜像：~30MB（Alpine 基础）
- 网络模式：`host`（容器直接访问 NAS 所有本地服务）

## 开发者

### 从源码编译

```bash
git clone https://github.com/ZhensJoke/fnos-frpc.git
cd fnos-frpc

# 全平台编译（Windows 下双击运行）
buildall.bat

# 仅编译 Docker 镜像
build.bat
```

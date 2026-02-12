# FNOS FRP 管理器

基于 Docker 的 FRP 客户端图形化配置工具，专为飞牛NAS (fnOS) 设计。

## 功能特性

- 🖥️ Web 图形界面管理 frpc 配置
- 📡 支持多个 frpc 服务器同时管理
- 🔄 支持 TCP/UDP/HTTP/HTTPS 代理类型
- 📦 frpc 在线安装（GitHub）和离线上传安装
- 🔒 首次使用设置管理密码
- 🐳 Docker 部署，host 网络模式

## 前置条件

- 一台安装了 fnOS 的 NAS（或任何支持 Docker 的 Linux 设备）
- NAS 已启用 Docker 功能
- 一台 Windows 电脑（用于编译打包，需安装 [Go](https://go.dev/dl/) 和 [Docker Desktop](https://www.docker.com/products/docker-desktop/)）

---

## 完整部署教程

### 第一步：在 Windows 上打包

1. **克隆项目**

```bash
git clone https://github.com/ZhensJoke/fnos-frpc.git
cd fnos-frpc
```

2. **一键打包**

确保 Docker Desktop 已启动，然后双击运行 `build.bat`。

脚本会自动完成：交叉编译 → 构建 Docker 镜像 → 导出为 `dist/fnos-frpc-gui.tar`。

打包完成后，`dist/` 目录下会有两个文件：

```
dist/
├── fnos-frpc-gui.tar    # Docker 镜像（~30MB）
└── docker-compose.yml   # 部署配置文件
```

### 第二步：上传文件到 NAS

将以下文件上传到 NAS 的同一个目录中（如 `/vol1/docker/frpc-gui/`）：

| 文件 | 来源 |
|------|------|
| `fnos-frpc-gui.tar` | `dist/` 目录 |
| `docker-compose.yml` | `dist/` 目录 |
| `deploy.sh` | 项目根目录 |

上传方式（任选其一）：
- **fnOS 文件管理器**：在 Web 界面中上传文件
- **SMB 共享**：通过 Windows 资源管理器拖拽到 NAS 共享文件夹
- **SCP 命令**：
  ```bash
  scp dist/fnos-frpc-gui.tar dist/docker-compose.yml deploy.sh user@NAS-IP:/vol1/docker/frpc-gui/
  ```

### 第三步：SSH 到 NAS 执行部署

1. **SSH 连接 NAS**

```bash
ssh user@NAS-IP
```

2. **进入文件目录**

```bash
cd /vol1/docker/frpc-gui/
```

3. **运行部署脚本**

```bash
chmod +x deploy.sh
./deploy.sh
```

脚本会自动：导入 Docker 镜像 → 创建数据目录 → 启动容器。

> 如果不使用脚本，也可以手动执行：
> ```bash
> docker load -i fnos-frpc-gui.tar
> mkdir -p data
> docker compose up -d
> ```

### 第四步：访问 Web 管理界面

浏览器打开：

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
|------|----|
| 名称 | `ssh` |
| 类型 | `TCP` |
| 本地 IP | `127.0.0.1` |
| 本地端口 | `22` |
| 远程端口 | `6022` |

**HTTP 代理示例（NAS Web 界面）：**

| 字段 | 值 |
|------|----|
| 名称 | `nas-web` |
| 类型 | `HTTP` |
| 本地 IP | `127.0.0.1` |
| 本地端口 | `5666` |
| 自定义域名 | `nas.yourdomain.com` |

### 5. 启动连接

在服务器详情页点击「启动」按钮，frpc 即开始运行。底部的日志区域会实时显示连接状态。

---

## 自定义端口

修改 `docker-compose.yml` 中的 `WEB_PORT` 环境变量：

```yaml
environment:
  - WEB_PORT=8080
```

然后重启容器：
```bash
docker compose down && docker compose up -d
```

## 数据持久化

配置数据保存在 `./data/` 目录中（Docker 卷挂载），包括：

| 文件/目录 | 内容 |
|----------|------|
| `auth.json` | 管理密码（bcrypt 哈希） |
| `servers.json` | 服务器和代理规则配置 |
| `frpc/` | frpc 二进制文件 |
| `conf/` | 自动生成的 frpc TOML 配置 |
| `logs/` | frpc 运行日志 |

> ⚠️ 备份 NAS 时建议一并备份 `data/` 目录。

## 更新升级

1. 在 Windows 上拉取最新代码并重新运行 `build.bat`
2. 将新的 `fnos-frpc-gui.tar` 上传到 NAS
3. 在 NAS 上执行：
```bash
docker compose down
docker load -i fnos-frpc-gui.tar
docker compose up -d
```

`data/` 目录中的配置不受影响，会自动保留。

## 技术栈

- 后端：Go（零外部依赖，仅标准库）
- 前端：HTML/CSS/JS（无框架）
- Docker 镜像：~30MB（Alpine 基础）
- 网络模式：`host`（容器直接访问 NAS 所有本地服务）

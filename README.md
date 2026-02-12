# FNOS FRP 管理器

基于 Docker 的 FRP 客户端图形化配置工具，专为飞牛NAS (fnOS) 设计。

## 功能特性

- 🖥️ Web 图形界面管理 frpc 配置
- 📡 支持多个 frpc 服务器同时管理
- 🔄 支持 TCP/UDP/HTTP/HTTPS 代理类型
- 📦 frpc 在线安装（GitHub）和离线上传安装
- 🔒 首次使用设置管理密码
- 🐳 Docker 部署，host 网络模式

## 快速部署

### 1. 构建并启动

```bash
docker-compose up -d --build
```

### 2. 访问管理界面

浏览器打开 `http://NAS-IP:7500`

### 3. 首次使用

1. 设置管理密码
2. 安装 frpc（在线下载或离线上传）
3. 添加 frps 服务器配置
4. 添加代理规则
5. 启动连接

## 自定义端口

修改 `docker-compose.yml` 中的 `WEB_PORT` 环境变量：

```yaml
environment:
  - WEB_PORT=8080
```

## 数据持久化

配置数据保存在 `./data` 目录中，包括：
- `servers.json` - 服务器和代理规则配置
- `auth.json` - 管理密码
- `frpc/` - frpc 二进制文件
- `conf/` - 生成的 frpc TOML 配置文件
- `logs/` - frpc 运行日志

## 技术栈

- 后端：Go（零外部依赖，仅标准库）
- 前端：HTML/CSS/JS（无框架）
- Docker 镜像：~30MB（Alpine 基础）

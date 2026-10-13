# My Panel

自用 Docker 容器管理面板。基于 [rabbit-panel](https://github.com/reisen7/rabbit-panel)（MIT）二次定制，Go 单二进制 + SQLite，内置前端，原生支持 `linux/amd64` 与 `linux/arm64`（含路由器 aarch64）。

## 功能

- 容器管理：列表 / 启动 / 停止 / 重启 / 强制停止 / 暂停 / 恢复 / 删除；可视化创建容器（端口、环境变量、挂载、网络、资源限制）；`docker run` 命令创建；容器终端与日志
- 镜像管理：列表 / 拉取（多架构选择）/ 导入 tar / 导出 tar / 删除 / 清理悬空镜像
- Compose 管理：项目列表 / 创建 / 编辑 / 启动 / 停止 / 重启 / 更新 / 删除 / 日志与终端
- 网络管理、存储卷管理、镜像仓库配置
- 文件管理、集成 Web 终端（xterm）、计划任务（Cron）
- 系统概览：CPU / 内存 / 磁盘与容器资源排行

## 技术栈

- 后端：Go (Gin) + Docker SDK + SQLite，单静态二进制
- 前端：Vue 3 + Vite + Element Plus，构建后由 Go `embed` 进二进制
- 部署：单容器，挂载 `/var/run/docker.sock`

---

## 一、首次发布到 GitHub

本仓库尚未关联远程。在项目根目录执行：

```bash
cd C:\Users\zghxi\Documents\GitHub\my-panel

git init
git add .
git commit -m "init: my panel from rabbit-panel"
git branch -M main

# 先到 GitHub 网页新建空仓库 fanling1970/my-panel（不要勾选 README）
git remote add origin https://github.com/fanling1970/my-panel.git
git push -u origin main
```

> 首次 push 后，GitHub Actions 会自动触发 `Build and Push Docker Image`，
> 构建多架构镜像并推送到 `ghcr.io/fanling1970/my-panel:latest`。

### 授权 GHCR（首次必做一次）

推送到 GitHub Container Registry 需要包写权限，本 workflow 已带 `packages: write`。
若 Actions 报 403：进入 GitHub 仓库 → Settings → Actions → General →
Workflow permissions → 选 **Read and write permissions** → Save。

镜像首次推送后，到 GitHub → 你的 profile → Packages → my-panel →
Package settings，确认它关联到 `my-panel` 仓库（设为 public 拉取无需登录）。

---

## 二、部署到路由器（ARM64）

在路由器上创建 `/root/my-panel/docker-compose.yml`，内容同本仓库
`docker-compose.deploy.yml`：

```bash
mkdir -p /root/my-panel/compose_projects /root/my-panel/data
cd /root/my-panel
# 把 docker-compose.deploy.yml 内容存为 docker-compose.yml
docker compose up -d
```

或直接 `docker run`：

```bash
docker run -d \
  --name my-panel \
  --restart unless-stopped \
  -p 3958:3958 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /root/my-panel/compose_projects:/app/compose_projects \
  -v /root/my-panel/data:/app/data \
  -e TZ=Asia/Shanghai \
  -e RABBIT_UPDATE_CHECK_DISABLED=true \
  ghcr.io/fanling1970/my-panel:latest
```

浏览器访问 `http://路由器IP:3958`，默认账号 **admin / admin**，首次登录后立即改密。

> 国内拉 GHCR 慢时，先在路由器配置 Docker 镜像加速器，或使用 ghcr 加速镜像地址。

---

## 三、日常更新

改完代码 push 到 main：

```bash
git add .
git commit -m "update: xxx"
git push
```

Actions 自动重新构建并覆盖 `:latest`。路由器上拉新版：

```bash
cd /root/my-panel
docker compose pull
docker compose up -d
```

如需固定版本，打 tag 后 Actions 会额外打版本标签：

```bash
git tag v0.1.0
git push origin v0.1.0
```

---

## 四、配置项（环境变量）

| 变量 | 默认 | 说明 |
|---|---|---|
| `PORT` | 3958 | 面板端口 |
| `JWT_SECRET` | 内置 | 登录令牌密钥，建议改成随机串 |
| `NODE_SECRET` | 内置 | 节点间通信密钥 |
| `TZ` | Asia/Shanghai | 时区 |
| `RABBIT_UPDATE_CHECK_DISABLED` | true | 禁用上游在线更新检查 |

## 许可

MIT，沿用上游 [rabbit-panel](https://github.com/reisen7/rabbit-panel)。

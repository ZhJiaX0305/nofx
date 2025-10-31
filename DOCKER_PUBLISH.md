# 🐳 Docker 镜像发布指南

本文档说明如何使用 Makefile 构建、标记和发布 NOFX Docker 镜像。

## 📋 前置要求

- Docker 已安装并运行
- 已登录 Docker Hub (`docker login`)
- 有权限推送到 `zhjiax/nofx` 仓库

## 🚀 快速开始

### 一键发布（推荐）

完整的构建、标记、上传流程：

```bash
make release
```

这个命令会自动执行：
1. 构建后端和前端镜像
2. 为镜像打标签
3. 推送到 Docker Hub

### 带版本号发布

```bash
make release VERSION=v1.0.0
```

这将创建以下标签：
- `zhjiax/nofx:backend`
- `zhjiax/nofx:frontend`
- `zhjiax/nofx:backend-v1.0.0`
- `zhjiax/nofx:frontend-v1.0.0`

## 📚 分步操作

### 1. 查看帮助信息

```bash
make help
```

### 2. 构建镜像

```bash
# 构建所有镜像
make build

# 只构建后端
make build-backend

# 只构建前端
make build-frontend
```

### 3. 标记镜像

```bash
# 标记所有镜像
make tag

# 标记后端镜像
make tag-backend

# 标记前端镜像
make tag-frontend

# 带版本号标记
make tag VERSION=v1.0.0
```

### 4. 推送镜像

```bash
# 推送所有镜像
make push

# 只推送后端
make push-backend

# 只推送前端
make push-frontend
```

### 5. 登录 Docker Hub

```bash
make login
```

### 6. 查看本地镜像

```bash
make list
```

### 7. 清理本地镜像

```bash
make clean
```

## 🔧 自定义配置

### 修改 Docker Hub 用户名

编辑 `Makefile` 中的变量：

```makefile
DOCKER_USERNAME = your_username
```

### 修改镜像名称

```makefile
IMAGE_NAME = your_image_name
```

### 修改标签名

```makefile
BACKEND_TAG = backend
FRONTEND_TAG = frontend
```

## 📦 发布工作流示例

### 开发版本发布

```bash
# 开发测试版本
make release VERSION=dev

# 测试版本
make release VERSION=beta
```

### 生产版本发布

```bash
# 发布 v1.0.0 版本
make release VERSION=v1.0.0

# 同时保留 latest 标签
make tag
make push
```

### 仅更新某个服务

```bash
# 只更新后端
make build-backend
make tag-backend
make push-backend

# 只更新前端
make build-frontend
make tag-frontend
make push-frontend
```

## 🌐 已发布的镜像

访问 Docker Hub 查看所有版本：
https://hub.docker.com/r/zhjiax/nofx

## 📥 使用已发布的镜像

其他用户可以通过以下方式使用：

### 拉取镜像

```bash
docker pull zhjiax/nofx:backend
docker pull zhjiax/nofx:frontend
```

### 直接运行

```bash
# 后端
docker run -d -p 8080:8080 \
  -v $(pwd)/config.json:/app/config.json:ro \
  zhjiax/nofx:backend

# 前端
docker run -d -p 3000:80 \
  zhjiax/nofx:frontend
```

### 使用 docker-compose

修改 `docker-compose.yml`，使用远程镜像：

```yaml
services:
  nofx:
    image: zhjiax/nofx:backend
    # 注释掉 build 部分
    ports:
      - "8080:8080"
    volumes:
      - ./config.json:/app/config.json:ro
    
  nofx-frontend:
    image: zhjiax/nofx:frontend
    # 注释掉 build 部分
    ports:
      - "3000:80"
    depends_on:
      - nofx
```

然后直接启动：

```bash
docker compose pull  # 拉取最新镜像
docker compose up -d
```

## 🔄 更新镜像

当代码更新后，重新发布：

```bash
# 方式 1: 覆盖 latest 标签
make release

# 方式 2: 发布新版本
make release VERSION=v1.0.1

# 方式 3: 先构建测试，确认无误后再推送
make build
# 测试镜像...
make tag VERSION=v1.0.1
make push
```

## ⚠️ 注意事项

1. **推送前先测试**：建议在推送前先本地测试镜像是否正常工作
2. **版本管理**：使用语义化版本号（如 v1.0.0, v1.1.0）
3. **latest 标签**：`latest` 标签始终指向最新发布的版本
4. **清理镜像**：定期运行 `make clean` 清理不需要的本地镜像

## 🐛 故障排查

### 推送失败

```bash
# 检查是否已登录
docker login

# 或使用 make 命令登录
make login
```

### 镜像构建失败

```bash
# 清理并重新构建
make clean
docker system prune -f
make build
```

### 查看构建日志

```bash
# 构建时查看详细输出
docker compose build --no-cache --progress=plain
```

## 📞 获取帮助

遇到问题？
- 运行 `make help` 查看所有可用命令
- 查看 [Docker 文档](https://docs.docker.com/)
- 提交 Issue 到项目仓库


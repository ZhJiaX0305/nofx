#!/bin/bash
# ================================================================
# 构建带版本号的 Docker 镜像
# 使用方法: ./build-version.sh <版本号>
# 示例: ./build-version.sh v2.1.0
# ================================================================

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

if [ -z "$1" ]; then
    echo -e "${RED}错误: 请提供版本号${NC}"
    echo "用法: $0 <版本号>"
    echo "示例: $0 v2.1.0"
    echo "      $0 v2024.11.03"
    echo "      $0 test-balance-check"
    exit 1
fi

VERSION=$1
DOCKER_USER="${DOCKER_USERNAME:-zhjiax}"
IMAGE_NAME="nofx"

echo "======================================"
echo "🏗️  构建版本镜像: $VERSION"
echo "======================================"
echo ""
echo "Docker Hub 用户: $DOCKER_USER"
echo "镜像名称: $IMAGE_NAME"
echo "版本标签: $VERSION"
echo ""

# 检查 buildx
echo -e "${YELLOW}检查 Docker Buildx...${NC}"
if ! docker buildx inspect multiplatform-builder &> /dev/null; then
    echo "创建 buildx builder..."
    docker buildx create --name multiplatform-builder --use
else
    echo "使用现有 builder..."
    docker buildx use multiplatform-builder
fi

docker buildx inspect --bootstrap

echo ""
echo -e "${YELLOW}构建并推送镜像...${NC}"
echo ""

# 构建后端镜像（AMD64）
echo -e "${GREEN}1/2 构建后端镜像...${NC}"
docker buildx build \
    --platform linux/amd64 \
    --file ./docker/Dockerfile.backend \
    --tag ${DOCKER_USER}/${IMAGE_NAME}:backend-${VERSION} \
    --tag ${DOCKER_USER}/${IMAGE_NAME}:backend-latest \
    --push \
    .

echo ""
echo -e "${GREEN}2/2 构建前端镜像...${NC}"
docker buildx build \
    --platform linux/amd64 \
    --file ./docker/Dockerfile.frontend \
    --tag ${DOCKER_USER}/${IMAGE_NAME}:frontend-${VERSION} \
    --tag ${DOCKER_USER}/${IMAGE_NAME}:frontend-latest \
    --push \
    .

echo ""
echo "======================================"
echo -e "${GREEN}✅ 版本镜像构建完成！${NC}"
echo "======================================"
echo ""
echo "已推送镜像（每个包含两个标签）："
echo "  ${DOCKER_USER}/${IMAGE_NAME}:backend-${VERSION}   (版本标签)"
echo "  ${DOCKER_USER}/${IMAGE_NAME}:backend-latest      (最新标签)"
echo "  ${DOCKER_USER}/${IMAGE_NAME}:frontend-${VERSION}  (版本标签)"
echo "  ${DOCKER_USER}/${IMAGE_NAME}:frontend-latest     (最新标签)"
echo ""
echo "💡 旧镜像仍然保留："
echo "  ${DOCKER_USER}/${IMAGE_NAME}:backend      (保持不变)"
echo "  ${DOCKER_USER}/${IMAGE_NAME}:frontend     (保持不变)"
echo ""
echo "🎯 三层标签体系："
echo "  backend-${VERSION}  ← 固定版本，永久保留"
echo "  backend-latest     ← 始终指向最新发布的版本"
echo "  backend            ← 保持不变（旧版本，用于生产稳定环境）"
echo ""
echo "📝 在 ECS 上使用："
echo ""
echo "方式 1: 使用 backend-latest（自动获取最新）"
echo "  修改 docker-compose.yml:"
echo "    image: ${DOCKER_USER}/${IMAGE_NAME}:backend-latest"
echo "  然后: docker compose pull && docker compose up -d"
echo ""
echo "方式 2: 使用固定版本号"
echo "  修改为: backend-${VERSION}"
echo ""
echo "方式 3: 保持旧版本（backend）"
echo "  不修改，继续使用 backend 标签"
echo ""


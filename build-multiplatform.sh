#!/bin/bash
# ================================================================
# 多平台 Docker 镜像构建脚本
# 支持 ARM64 (Mac M1/M2) 和 AMD64 (x86_64 服务器)
# ================================================================

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "======================================"
echo "🏗️  多平台 Docker 镜像构建"
echo "======================================"
echo ""

# 从环境变量获取 Docker Hub 用户名，如果未设置则使用默认值
DOCKER_USER="${DOCKER_USERNAME:-zhjiax}"
IMAGE_NAME="nofx"

echo "Docker Hub 用户: $DOCKER_USER"
echo "镜像名称: $IMAGE_NAME"
echo ""

echo -e "${YELLOW}检查 Docker Buildx...${NC}"
# 创建并使用 buildx builder
if ! docker buildx inspect multiplatform-builder &> /dev/null; then
    echo "创建 buildx builder..."
    docker buildx create --name multiplatform-builder --use
else
    echo "使用现有 builder..."
    docker buildx use multiplatform-builder
fi

# 启动 builder
docker buildx inspect --bootstrap

echo ""
echo -e "${YELLOW}构建并推送多平台镜像...${NC}"
echo ""

# 构建后端镜像
echo -e "${GREEN}1/2 构建后端镜像 (ARM64 + AMD64)...${NC}"
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    --file ./docker/Dockerfile.backend \
    --tag ${DOCKER_USER}/${IMAGE_NAME}:backend \
    --tag ${DOCKER_USER}/${IMAGE_NAME}:backend-latest \
    --push \
    .

echo ""
echo -e "${GREEN}2/2 构建前端镜像 (ARM64 + AMD64)...${NC}"
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    --file ./docker/Dockerfile.frontend \
    --tag ${DOCKER_USER}/${IMAGE_NAME}:frontend \
    --tag ${DOCKER_USER}/${IMAGE_NAME}:frontend-latest \
    --push \
    .

echo ""
echo "======================================"
echo -e "${GREEN}✅ 多平台镜像构建完成！${NC}"
echo "======================================"
echo ""
echo "已推送镜像："
echo "  ${DOCKER_USER}/${IMAGE_NAME}:backend (amd64 + arm64)"
echo "  ${DOCKER_USER}/${IMAGE_NAME}:frontend (amd64 + arm64)"
echo ""
echo "现在可以在任何平台上拉取和运行这些镜像！"
echo ""


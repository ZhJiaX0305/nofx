#!/bin/bash
# ================================================================
# 只构建 AMD64 平台镜像（用于 ECS 部署）
# 比多平台构建快很多
# ================================================================

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "======================================"
echo "🏗️  AMD64 平台镜像构建"
echo "======================================"
echo ""

# 从环境变量获取 Docker Hub 用户名，如果未设置则使用默认值
DOCKER_USER="${DOCKER_USERNAME:-zhjiax}"
IMAGE_NAME="nofx"

echo "Docker Hub 用户: $DOCKER_USER"
echo "镜像名称: $IMAGE_NAME"
echo ""

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
echo -e "${YELLOW}只构建 AMD64 平台镜像（更快）...${NC}"
echo ""

# 构建后端镜像（只支持 AMD64）
echo -e "${GREEN}1/2 构建后端镜像 (AMD64 only)...${NC}"
docker buildx build \
    --platform linux/amd64 \
    --file ./docker/Dockerfile.backend \
    --tag ${DOCKER_USER}/${IMAGE_NAME}:backend \
    --push \
    .

echo ""
echo -e "${GREEN}2/2 构建前端镜像 (AMD64 only)...${NC}"
docker buildx build \
    --platform linux/amd64 \
    --file ./docker/Dockerfile.frontend \
    --tag ${DOCKER_USER}/${IMAGE_NAME}:frontend \
    --push \
    .

echo ""
echo "======================================"
echo -e "${GREEN}✅ AMD64 镜像构建完成！${NC}"
echo "======================================"
echo ""
echo "已推送镜像："
echo "  ${DOCKER_USER}/${IMAGE_NAME}:backend (amd64)"
echo "  ${DOCKER_USER}/${IMAGE_NAME}:frontend (amd64)"
echo ""
echo "⚠️  注意：这些镜像只能在 x86_64 服务器上运行"
echo "    如果需要在 Mac M1/M2 上运行，请使用 build-multiplatform.sh"
echo ""


#!/bin/bash
# ================================================================
# 只构建和推送后端镜像（快速更新）
# 使用方法: ./build-backend.sh [版本号]
# ================================================================

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

VERSION=${1:-""}
DOCKER_USER="${DOCKER_USERNAME:-zhjiax}"
IMAGE_NAME="nofx"

echo "======================================"
echo "🏗️  构建后端镜像"
echo "======================================"
echo ""
echo "Docker Hub 用户: $DOCKER_USER"
echo "镜像名称: $IMAGE_NAME"
echo ""
echo -e "${YELLOW}构建后端镜像...${NC}"

# 根据是否提供版本号决定标签
if [ -n "$VERSION" ]; then
    echo "版本: $VERSION"
    echo ""
    
    docker buildx build \
        --platform linux/amd64 \
        --file ./docker/Dockerfile.backend \
        --tag ${DOCKER_USER}/${IMAGE_NAME}:backend-${VERSION} \
        --tag ${DOCKER_USER}/${IMAGE_NAME}:backend-latest \
        --push \
        .
    
    echo ""
    echo "======================================"
    echo -e "${GREEN}✅ 后端镜像构建完成！${NC}"
    echo "======================================"
    echo ""
    echo "已推送镜像："
    echo "  ${DOCKER_USER}/${IMAGE_NAME}:backend-${VERSION}"
    echo "  ${DOCKER_USER}/${IMAGE_NAME}:backend-latest"
    echo ""
else
    echo "使用默认标签 (backend)"
    echo ""
    
    docker buildx build \
        --platform linux/amd64 \
        --file ./docker/Dockerfile.backend \
        --tag ${DOCKER_USER}/${IMAGE_NAME}:backend \
        --push \
        .
    
    echo ""
    echo "======================================"
    echo -e "${GREEN}✅ 后端镜像构建完成！${NC}"
    echo "======================================"
    echo ""
    echo "已推送镜像："
    echo "  ${DOCKER_USER}/${IMAGE_NAME}:backend"
    echo ""
fi

echo "🚀 在 ECS 上更新："
echo "  ssh root@<ECS_IP>"
echo "  cd /root/workspace/nofx"
echo "  docker compose pull nofx"
echo "  docker compose up -d nofx"
echo ""


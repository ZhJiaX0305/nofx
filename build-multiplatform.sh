#!/bin/bash
# ================================================================
# 平台特定 Docker 镜像构建脚本
# 使用方法: ./build-multiplatform.sh [local|prod]
#   local - 构建 ARM64 (Mac M1/M2)
#   prod  - 构建 AMD64 (x86_64 服务器)
# ================================================================

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# 获取平台参数（必须指定）
if [ -z "$1" ]; then
    echo -e "${RED}错误: 必须指定平台参数${NC}"
    echo ""
    echo "用法: $0 [local|prod]"
    echo ""
    echo "参数说明:"
    echo "  local - 构建 ARM64 (Mac M1/M2 本地开发)"
    echo "  prod  - 构建 AMD64 (ECS 生产服务器)"
    echo ""
    echo "示例:"
    echo "  $0 local  # 构建 ARM64 镜像"
    echo "  $0 prod   # 构建 AMD64 镜像"
    echo ""
    echo "💡 提示:"
    echo "  - 本地开发推荐使用: ./dev-local.sh (不推送镜像)"
    echo "  - ECS 部署推荐使用: ./build-multiplatform.sh prod"
    exit 1
fi

PLATFORM_MODE=$1

# 从环境变量获取 Docker Hub 用户名，如果未设置则使用默认值
DOCKER_USER="${DOCKER_USERNAME:-zhjiax}"
IMAGE_NAME="nofx"

# 根据参数确定构建平台、标签和推送选项
case "$PLATFORM_MODE" in
    local)
        PLATFORMS="linux/arm64"
        PLATFORM_DESC="ARM64 (Mac M1/M2)"
        TAG_SUFFIX="local"  # 本地标签
        PUSH_FLAG=""  # 本地不推送
        OUTPUT_TYPE="--load"  # 加载到本地 Docker
        ;;
    prod)
        PLATFORMS="linux/amd64"
        PLATFORM_DESC="AMD64 (x86_64 服务器)"
        TAG_SUFFIX="latest"  # 生产标签
        PUSH_FLAG="--push"  # 生产推送到 Docker Hub
        OUTPUT_TYPE=""
        ;;
    *)
        echo -e "${RED}错误: 无效的平台参数 '$PLATFORM_MODE'${NC}"
        echo ""
        echo "用法: $0 [local|prod]"
        echo ""
        echo "参数说明:"
        echo "  local - 构建 ARM64 (本地使用，不推送)"
        echo "  prod  - 构建 AMD64 (推送到 Docker Hub)"
        echo ""
        exit 1
        ;;
esac

echo "======================================"
echo "🏗️  Docker 镜像构建"
echo "======================================"
echo ""
echo "Docker Hub 用户: $DOCKER_USER"
echo "镜像名称: $IMAGE_NAME"
echo "构建平台: $PLATFORM_DESC"
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
echo -e "${GREEN}1/2 构建后端镜像 ($PLATFORM_DESC)...${NC}"
docker buildx build \
    --platform ${PLATFORMS} \
    --file ./docker/Dockerfile.backend \
    --tag ${DOCKER_USER}/${IMAGE_NAME}:backend-${TAG_SUFFIX} \
    ${OUTPUT_TYPE} \
    ${PUSH_FLAG} \
    .

echo ""
echo -e "${GREEN}2/2 构建前端镜像 ($PLATFORM_DESC)...${NC}"
docker buildx build \
    --platform ${PLATFORMS} \
    --file ./docker/Dockerfile.frontend \
    --tag ${DOCKER_USER}/${IMAGE_NAME}:frontend-${TAG_SUFFIX} \
    ${OUTPUT_TYPE} \
    ${PUSH_FLAG} \
    .

echo ""
echo "======================================"
echo -e "${GREEN}✅ 镜像构建完成！${NC}"
echo "======================================"
echo ""

case "$PLATFORM_MODE" in
    local)
        echo "已构建本地镜像 ($PLATFORM_DESC)："
        echo "  ${DOCKER_USER}/${IMAGE_NAME}:backend-local"
        echo "  ${DOCKER_USER}/${IMAGE_NAME}:frontend-local"
        echo ""
        echo "🍎 本地使用:"
        echo "  docker compose up -d"
        echo "  或: ./dev-local.sh start"
        echo ""
        echo "💡 镜像已加载到本地 Docker，未推送到远程仓库"
        ;;
    prod)
        echo "已推送生产镜像 ($PLATFORM_DESC)："
        echo "  ${DOCKER_USER}/${IMAGE_NAME}:backend-latest"
        echo "  ${DOCKER_USER}/${IMAGE_NAME}:frontend-latest"
        echo ""
        echo "☁️  镜像已推送到 Docker Hub"
        echo "   查看: https://hub.docker.com/r/${DOCKER_USER}/${IMAGE_NAME}/tags"
        echo ""
        ;;
esac

echo ""


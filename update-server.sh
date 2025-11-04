#!/bin/bash
# ECS 上更新 NOFX 服务脚本
# 用法:
#   ./update-server.sh all              # 更新所有服务（排除 nginx-proxy-manager）
#   ./update-server.sh nofx             # 只更新后端
#   ./update-server.sh nofx-frontend    # 只更新前端
# ================================================================

cd /root/workspace/nofx

SERVICE=$1

# 参数校验
if [ -z "$SERVICE" ]; then
  echo "❌ 错误：必须指定要更新的服务"
  echo ""
  echo "用法:"
  echo "  $0 all              # 更新所有服务（后端 + 前端，排除 nginx-proxy-manager）"
  echo "  $0 nofx             # 只更新后端服务"
  echo "  $0 nofx-frontend    # 只更新前端服务"
  echo ""
  exit 1
fi

echo "======================================"
echo "🔄  更新 NOFX 服务"
echo "======================================"
echo ""

if [ "$SERVICE" = "all" ]; then
  echo "📦  拉取最新镜像（排除 nginx-proxy-manager）..."
  docker compose pull nofx nofx-frontend
  
  echo "🚀  更新业务容器..."
  docker compose up -d nofx nofx-frontend
else
  echo "📦  拉取 $SERVICE 镜像..."
  docker compose pull $SERVICE
  
  echo "🚀  更新 $SERVICE 容器..."
  docker compose up -d $SERVICE
fi

# 等待启动
echo "⏳  等待服务启动..."
sleep 5

# 显示状态
echo ""
echo "✅  更新完成！"
echo ""
echo "📊  服务状态:"
docker compose ps

# 清理旧镜像
echo ""
echo "🧹  清理未使用的镜像..."
docker image prune -f

echo ""
echo "💡  查看日志:"
if [ -z "$SERVICE" ]; then
  echo "    所有服务: docker compose logs -f"
  echo "    后端服务: docker compose logs -f nofx"
  echo "    前端服务: docker compose logs -f nofx-frontend"
else
  echo "    docker compose logs -f $SERVICE"
fi
echo ""
echo "📝  提示: nginx-proxy-manager 默认不会更新，如需更新请手动执行:"
echo "    docker compose pull nginx-proxy-manager"
echo "    docker compose up -d nginx-proxy-manager"


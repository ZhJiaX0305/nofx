#!/bin/bash
# ================================================================
# NOFX 本地开发快速启动脚本 (Mac M1/M2)
# ================================================================

cd /Users/alanwork/GitHub/nofx

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 确保数据库文件存在（避免Docker创建目录）
ensure_database() {
    if [ -d "config.db" ]; then
        echo -e "${YELLOW}⚠️  config.db 是目录，正在删除...${NC}"
        rm -rf config.db
    fi
    
    if [ ! -f "config.db" ]; then
        echo -e "${YELLOW}📋 创建空数据库文件...${NC}"
        touch config.db
        echo -e "${GREEN}✓ 已创建空数据库文件${NC}"
    fi
}

case "$1" in
    start)
        echo -e "${YELLOW}🚀 启动本地开发环境...${NC}"
        ensure_database
        docker compose -f docker-compose.local.yml --build
        docker compose -f docker-compose.local.yml up -d
        echo ""
        echo -e "${GREEN}✅ 服务已启动${NC}"
        echo ""
        echo "📊 服务状态:"
        docker compose -f docker-compose.local.yml ps
        echo ""
        echo "🌐 访问地址:"
        echo "  Web 界面: http://localhost:3000"
        echo "  API 服务: http://localhost:8080"
        echo ""
        echo "📋 查看日志: ./dev-local.sh logs"
        ;;
    
    stop)
        echo -e "${YELLOW}⏹ 停止本地服务...${NC}"
        docker compose -f docker-compose.local.yml stop
        echo -e "${GREEN}✅ 服务已停止${NC}"
        ;;
    
    restart)
        echo -e "${YELLOW}🔄 重启本地服务...${NC}"
        docker compose -f docker-compose.local.yml restart
        echo -e "${GREEN}✅ 服务已重启${NC}"
        docker compose -f docker-compose.local.yml ps
        ;;
    
    rebuild)
        echo -e "${YELLOW}🏗️  重新构建并启动...${NC}"
        ensure_database
        docker compose -f docker-compose.local.yml up -d --build --force-recreate
        echo -e "${GREEN}✅ 重新构建完成${NC}"
        ;;
    
    logs)
        echo -e "${YELLOW}📋 查看日志（Ctrl+C 退出）...${NC}"
        docker compose -f docker-compose.local.yml logs -f
        ;;
    
    status)
        echo "📊 服务状态:"
        docker compose -f docker-compose.local.yml ps
        echo ""
        echo "💾 资源使用:"
        docker stats --no-stream nofx-trading-local nofx-frontend-local 2>/dev/null || echo "容器未运行"
        ;;
    
    clean)
        echo -e "${YELLOW}🧹 清理本地环境...${NC}"
        docker compose -f docker-compose.local.yml down
        echo -e "${GREEN}✅ 清理完成${NC}"
        ;;
    
    *)
        echo "NOFX 本地开发工具"
        echo ""
        echo "用法: $0 {start|stop|restart|rebuild|logs|status|clean}"
        echo ""
        echo "命令:"
        echo "  start   - 启动服务（自动构建）"
        echo "  stop    - 停止服务"
        echo "  restart - 重启服务（不重新构建）"
        echo "  rebuild - 重新构建并启动"
        echo "  logs    - 查看实时日志"
        echo "  status  - 查看服务状态"
        echo "  clean   - 停止并删除容器"
        echo ""
        echo "💡 提示:"
        echo "  - 修改 config.json/prompts 后: ./dev-local.sh restart"
        echo "  - 修改 Go 代码后: ./dev-local.sh rebuild"
        echo "  - 修改前端代码后: ./dev-local.sh rebuild"
        exit 1
        ;;
esac


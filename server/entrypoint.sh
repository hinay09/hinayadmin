#!/bin/sh
# Docker 启动入口脚本
# 在启动 Go 应用前, 用环境变量替换配置文件中的占位符
# GoFrame 不支持 YAML 配置文件中的环境变量替换, 因此需要此脚本

set -e

CONFIG_FILE="/app/manifest/config/config.docker.yaml"

# 必填环境变量前置校验: 缺失时快速失败, 避免带占位符配置启动
if [ -z "$JWT_SECRET" ]; then
    echo "ERROR: JWT_SECRET 环境变量未设置, 拒绝启动 (生成: openssl rand -hex 32)" >&2
    exit 1
fi
if [ -z "$MYSQL_PASSWORD" ]; then
    echo "ERROR: MYSQL_PASSWORD 环境变量未设置, 拒绝启动 (compose 部署由 .env 的 MYSQL_ROOT_PASSWORD 注入)" >&2
    exit 1
fi

# 连接参数默认值与 docker-compose 内置服务一致; 使用外部 MySQL/Redis 时经环境变量覆盖
MYSQL_HOST="${MYSQL_HOST:-mysql}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_DATABASE="${MYSQL_DATABASE:-hinay_admin}"
REDIS_HOST="${REDIS_HOST:-redis}"
REDIS_PORT="${REDIS_PORT:-6379}"
DEMO_MODE="${DEMO_MODE:-false}"

if [ -f "$CONFIG_FILE" ]; then
    # 数据库连接
    sed -i "s|\${MYSQL_HOST}|${MYSQL_HOST}|g" "$CONFIG_FILE"
    sed -i "s|\${MYSQL_PORT}|${MYSQL_PORT}|g" "$CONFIG_FILE"
    sed -i "s|\${MYSQL_USER}|${MYSQL_USER}|g" "$CONFIG_FILE"
    sed -i "s|\${MYSQL_DATABASE}|${MYSQL_DATABASE}|g" "$CONFIG_FILE"
    sed -i "s|\${MYSQL_PASSWORD}|${MYSQL_PASSWORD}|g" "$CONFIG_FILE"

    # Redis 连接
    sed -i "s|\${REDIS_HOST}|${REDIS_HOST}|g" "$CONFIG_FILE"
    sed -i "s|\${REDIS_PORT}|${REDIS_PORT}|g" "$CONFIG_FILE"

    # 替换 JWT 密钥占位符
    sed -i "s|\${JWT_SECRET}|${JWT_SECRET}|g" "$CONFIG_FILE"

    # 替换演示模式开关占位符
    sed -i "s|\${DEMO_MODE}|${DEMO_MODE}|g" "$CONFIG_FILE"
fi

exec /app/main

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
if [ -z "$MYSQL_ROOT_PASSWORD" ]; then
    echo "ERROR: MYSQL_ROOT_PASSWORD 环境变量未设置, 拒绝启动" >&2
    exit 1
fi

if [ -f "$CONFIG_FILE" ]; then
    # 替换数据库密码占位符
    sed -i "s|\${MYSQL_ROOT_PASSWORD}|${MYSQL_ROOT_PASSWORD}|g" "$CONFIG_FILE"

    # 替换 JWT 密钥占位符
    sed -i "s|\${JWT_SECRET}|${JWT_SECRET}|g" "$CONFIG_FILE"
fi

exec /app/main

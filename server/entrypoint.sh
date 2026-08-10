#!/bin/sh
# Docker 启动入口脚本
# 在启动 Go 应用前, 用环境变量替换配置文件中的占位符
# GoFrame 不支持 YAML 配置文件中的环境变量替换, 因此需要此脚本

set -e

CONFIG_FILE="/app/manifest/config/config.docker.yaml"

if [ -f "$CONFIG_FILE" ]; then
    # 替换数据库密码占位符
    if [ -n "$MYSQL_ROOT_PASSWORD" ]; then
        sed -i "s|\${MYSQL_ROOT_PASSWORD}|${MYSQL_ROOT_PASSWORD}|g" "$CONFIG_FILE"
    fi

    # 替换 JWT 密钥占位符
    if [ -n "$JWT_SECRET" ]; then
        sed -i "s|\${JWT_SECRET}|${JWT_SECRET}|g" "$CONFIG_FILE"
    fi
fi

exec /app/main

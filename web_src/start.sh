#!/bin/sh
# 容器入口: 启动 nginx(前台) 与 Nuxt SSR, 并正确转发停止信号
nginx -g "daemon off;" &
NGINX_PID=$!
node .output/server/index.mjs &
NODE_PID=$!

shutdown() {
    kill "$NGINX_PID" "$NODE_PID" 2>/dev/null
    wait "$NGINX_PID" "$NODE_PID" 2>/dev/null
    exit 0
}
trap shutdown TERM INT

wait -n "$NGINX_PID" "$NODE_PID"
shutdown

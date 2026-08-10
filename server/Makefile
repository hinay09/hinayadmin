ROOT_DIR    = $(shell pwd)
NAMESPACE   = "default"
DEPLOY_NAME = "hinay-admin"
DOCKER_NAME = "hinay-admin"

include ./hack/hack-cli.mk
include ./hack/hack.mk

# ===== Hinay Admin 自定义命令 =====

# 本地运行后端 (开发模式)
.PHONY: run
run:
	@go run main.go

# 初始化数据库 (请在 manifest/config/config.yaml 中配好 MySQL 连接)
# 默认: make initdb DB_USER=root DB_PASS=hinay_admin DB_NAME=hinay_admin DB_HOST=127.0.0.1 DB_PORT=3306
# 覆写: make initdb DB_USER=xxx DB_PASS=xxx DB_NAME=xxx
DB_USER ?= root
DB_PASS ?= hinay_admin
DB_NAME ?= hinay_admin
DB_HOST ?= 127.0.0.1
DB_PORT ?= 3306

.PHONY: initdb
initdb:
	@echo ">> 确保数据库 $(DB_NAME) 存在"
	@mysql -u$(DB_USER) -p'$(DB_PASS)' -h$(DB_HOST) -P$(DB_PORT) -e "CREATE DATABASE IF NOT EXISTS $(DB_NAME) DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci;"
	@echo ">> 执行 manifest/sql/init.sql"
	@mysql -u$(DB_USER) -p'$(DB_PASS)' -h$(DB_HOST) -P$(DB_PORT) $(DB_NAME) < manifest/sql/init.sql
	@echo ">> 数据库初始化完成"

# 仅编译验证
.PHONY: vet
vet:
	@go vet ./...

# 代码格式化
.PHONY: fmt
fmt:
	@gofmt -s -w .

# 清理依赖
.PHONY: tidy
tidy:
	@go mod tidy

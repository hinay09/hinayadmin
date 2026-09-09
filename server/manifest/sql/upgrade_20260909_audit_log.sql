-- 升级脚本: 操作日志表增强(2026-09)
-- 适用于以旧版 init.sql 初始化的库, 增加请求上下文与结果列, 便于日志排查
USE `hinay_admin`;

ALTER TABLE `sys_audit_log`
  ADD COLUMN `method`      VARCHAR(10)  NOT NULL DEFAULT '' COMMENT 'HTTP方法'        AFTER `resource_id`,
  ADD COLUMN `path`        VARCHAR(255) NOT NULL DEFAULT '' COMMENT '请求路径'        AFTER `method`,
  ADD COLUMN `status_code` INT          NOT NULL DEFAULT 0  COMMENT 'HTTP状态码'      AFTER `path`,
  ADD COLUMN `code`        INT          NOT NULL DEFAULT 0  COMMENT '业务码(0=成功)'  AFTER `status_code`,
  ADD COLUMN `message`     VARCHAR(512) NOT NULL DEFAULT '' COMMENT '业务消息/失败原因' AFTER `code`,
  ADD COLUMN `duration_ms` INT          NOT NULL DEFAULT 0  COMMENT '耗时(毫秒)'      AFTER `message`,
  ADD COLUMN `request_id`  VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '请求ID(链路追踪)' AFTER `duration_ms`,
  ADD INDEX `idx_code` (`code`);

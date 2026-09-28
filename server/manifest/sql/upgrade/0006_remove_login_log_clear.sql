-- ============================================================
-- 移除登录日志"清空"功能 (增量脚本: 适用于已执行过 0001_login_log.sql 的存量库)
-- 说明: 清空 = 无 WHERE 全表 DELETE, 与 GoFrame 防误删保护冲突且风险高,
--       该功能已下线, 登录日志仅支持按 ID 逐条删除。
-- 全新安装的库不包含这些种子, 无需执行本文件。
-- 执行方式: mysql -u<user> -p <库名> < manifest/sql/upgrade/0006_remove_login_log_clear.sql
-- ============================================================

-- 移除"登录日志清空"按钮菜单 (id=182)
DELETE FROM `sys_menu` WHERE `id` = 182 AND `permission` = 'system:login-log:clear';

-- 移除对应 Casbin 策略
DELETE FROM `casbin_rule` WHERE `ptype` = 'p' AND `v1` = 'menu:182';

-- 移除"清空登录日志"API 资源 (精确路径, 不含按 ID 删除的 /:id)
DELETE FROM `sys_api`
WHERE `path` = '/api/v1/system/login-logs'
  AND `method` = 'DELETE'
  AND `description` = '清空登录日志';

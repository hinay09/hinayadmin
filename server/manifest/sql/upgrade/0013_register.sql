-- ============================================================
-- 0013: 开放注册 (增量升级脚本: 适用于已按旧版 init.sql 初始化的存量库)
-- 全新安装直接执行 init.sql, 无需本文件。
-- 执行方式: mysql -u<user> -p <库名> < manifest/sql/upgrade/0013_register.sql
-- 功能: sys.allow_register 已有配置项, 本脚本仅补注册接口的 sys_api 资源记录
--       (接口为公开路径, 不入 Casbin 策略; 记录仅用于接口管理页展示)。
-- ============================================================

INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/auth/register',        'POST', '认证', '用户注册(公开, 受 sys.allow_register 开关控制)'),
  ('/api/v1/auth/register/status', 'GET',  '认证', '查询注册开关(公开)');

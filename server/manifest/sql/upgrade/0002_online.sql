-- ============================================================
-- 在线用户 (增量升级脚本: 适用于已按旧版 init.sql 初始化的存量库)
-- 全新安装直接执行 init.sql, 无需本文件。
-- 会话数据存 Redis (hinay:online:sessions), 无需建表, 仅种子数据。
-- 执行方式: mysql -u<user> -p <库名> < manifest/sql/upgrade/0002_online.sql
-- ============================================================

-- 菜单: 在线用户 (顶级, 与操作日志/登录日志并列)
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (19, 0, '在线用户', 2, '/system/online', 'system/online/index', 'Monitor', 'system:online:list', 101, 1, 1);

-- 按钮权限
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (191, 19, '强制下线', 3, '', '', '', 'system:online:kick', 1, 0, 1);

-- API 资源
INSERT INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/online',     'GET',    '在线用户', '在线用户列表'),
  ('/api/v1/system/online/:id', 'DELETE', '在线用户', '强制下线');

-- Casbin: admin 角色
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', 'admin', 'menu:19',  'access', '', '', ''),
  ('p', 'admin', 'menu:191', 'access', '', '', '');

-- ============================================================
-- 菜单合并 (增量升级脚本: 适用于已按旧版 init.sql 初始化的存量库)
-- 全新安装直接执行 init.sql, 无需本文件。
--
-- 变更: 顶级菜单 9 个 -> 5 个, 近期新增功能归入两个新目录
--   系统监控 (id=2): 在线用户 / 定时任务 / 登录日志 / 操作日志
--   系统工具 (id=3): 代码生成
-- 子菜单 path / permission / 按钮 / API 资源均不变, 仅调整归属与排序。
-- 执行方式: mysql -u<user> -p <库名> < manifest/sql/upgrade/0010_menu_merge.sql
-- 注意: 一次性脚本, 重复执行会因主键/唯一键冲突报错; 执行后重新登录(或刷新)生效。
-- ============================================================

-- 1. 新增两个目录
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (2, 0, '系统监控', 1, '/monitor', 'Layout', 'Monitor', 'monitor', 20, 1, 1),
  (3, 0, '系统工具', 1, '/tool',    'Layout', 'Tools',   'tool',    25, 1, 1);

-- 2. 归档: 五个原顶级菜单挂入目录 (按钮权限 parent 不变, 无需处理)
UPDATE `sys_menu` SET `parent_id` = 2, `sort` = 1, `icon` = 'View' WHERE `id` = 19; -- 在线用户(图标改 View, 避免与目录 Monitor 重复)
UPDATE `sys_menu` SET `parent_id` = 2, `sort` = 2 WHERE `id` = 20; -- 定时任务
UPDATE `sys_menu` SET `parent_id` = 2, `sort` = 3 WHERE `id` = 18; -- 登录日志
UPDATE `sys_menu` SET `parent_id` = 2, `sort` = 4 WHERE `id` = 16; -- 操作日志
UPDATE `sys_menu` SET `parent_id` = 3, `sort` = 1 WHERE `id` = 21; -- 代码生成

-- 3. admin 角色目录授权 (超管走全局放行, 补齐保持种子完整)
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', 'admin', 'menu:2', 'access', '', '', ''),
  ('p', 'admin', 'menu:3', 'access', '', '', '');

-- 4. 非超管角色: 菜单树自顶向下构建, 子菜单须父目录可见才渲染,
--    故为所有持有被移动菜单的角色自动补授对应目录(含 admin, 已持有则跳过), 避免子菜单"失联"。
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p', r.`v0`, 'menu:2', 'access', '', '', ''
FROM (
  SELECT DISTINCT `v0` FROM `casbin_rule`
  WHERE `ptype` = 'p' AND `v2` = 'access'
    AND `v1` IN ('menu:16', 'menu:18', 'menu:19', 'menu:20', 'menu:201', 'menu:202', 'menu:203',
                 'menu:204', 'menu:205', 'menu:206', 'menu:181', 'menu:191')
) r
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` e
  WHERE e.`ptype` = 'p' AND e.`v0` = r.`v0` AND e.`v1` = 'menu:2' AND e.`v2` = 'access'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p', r.`v0`, 'menu:3', 'access', '', '', ''
FROM (
  SELECT DISTINCT `v0` FROM `casbin_rule`
  WHERE `ptype` = 'p' AND `v2` = 'access'
    AND `v1` IN ('menu:21', 'menu:211', 'menu:212', 'menu:213')
) r
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` e
  WHERE e.`ptype` = 'p' AND e.`v0` = r.`v0` AND e.`v1` = 'menu:3' AND e.`v2` = 'access'
);

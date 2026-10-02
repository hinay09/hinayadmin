-- ============================================================
-- Casbin 关联键 ID 化 (增量升级脚本: 适用于已按旧版 init.sql 初始化的存量库)
-- 全新安装直接执行 init.sql, 无需本文件。
--
-- 变更: casbin_rule 的关联键从 用户名字符串/角色code字符串 迁移为 用户ID/角色ID
--   g 行: (g, username, 角色code) -> (g, 用户ID, 角色ID)
--   p 行: (p, 角色code, 资源, 操作) -> (p, 角色ID, 资源, 操作)
-- 迁移后修改用户名或角色 code 不再影响任何权限功能, 角色 code 降级为纯展示标识。
--
-- 执行方式: mysql -u<user> -p <库名> < manifest/sql/upgrade/0012_casbin_id_subject.sql
-- 注意:
--   1. 执行后需重启服务 (casbin enforcer 启动时全量加载), 或等待下次策略变更触发 Reload。
--   2. 本脚本可安全重复执行: 二次运行时字符串 JOIN 不再命中, 各语句均为空操作。
--   3. 仅当存在用户名/角色 code 与 ID 字符串撞车的脏数据 (如用户名恰为纯数字 "1") 时
--      可能报唯一键冲突, 需人工处理该行后再执行。
--   4. 孤儿行 (用户/角色记录已物理缺失的 g/p 行) 会被清理; 软删用户的行保留, 无害。
-- ============================================================

-- 1. g 行: username -> 用户ID, 角色code -> 角色ID
UPDATE `casbin_rule` c
  JOIN `sys_user` u ON u.`username` = c.`v0`
  JOIN `sys_role` r ON r.`code` = c.`v1`
  SET c.`v0` = u.`id`, c.`v1` = r.`id`
  WHERE c.`ptype` = 'g';

-- 2. p 行: 角色code -> 角色ID
UPDATE `casbin_rule` c
  JOIN `sys_role` r ON r.`code` = c.`v0`
  SET c.`v0` = r.`id`
  WHERE c.`ptype` = 'p';

-- 3. 清理孤儿行 (JOIN 未命中的行仍保留字符串键, 按 ID 关联不上即删除)
DELETE c FROM `casbin_rule` c
  LEFT JOIN `sys_user` u ON u.`id` = c.`v0`
  WHERE c.`ptype` = 'g' AND u.`id` IS NULL;
DELETE c FROM `casbin_rule` c
  LEFT JOIN `sys_role` r ON r.`id` = c.`v1`
  WHERE c.`ptype` = 'g' AND r.`id` IS NULL;
DELETE c FROM `casbin_rule` c
  LEFT JOIN `sys_role` r ON r.`id` = c.`v0`
  WHERE c.`ptype` = 'p' AND r.`id` IS NULL;

-- 4. 列注释对齐新语义
ALTER TABLE `casbin_rule`
  MODIFY `v0` VARCHAR(100) NOT NULL DEFAULT '' COMMENT 'p:sub(角色ID) | g:用户ID',
  MODIFY `v1` VARCHAR(100) NOT NULL DEFAULT '' COMMENT 'p:obj(path/menu:菜单ID) | g:角色ID';

-- 5. 校验 (人工执行): 不应再有非纯数字的关联键
-- SELECT ptype, v0, v1 FROM casbin_rule
--  WHERE (ptype='g' AND (v0 NOT REGEXP '^[0-9]+$' OR v1 NOT REGEXP '^[0-9]+$'))
--     OR (ptype='p' AND v0 NOT REGEXP '^[0-9]+$');

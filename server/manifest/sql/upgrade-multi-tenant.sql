-- ============================================================
-- 多租户改造 存量库一次性升级脚本 (单租户 → 多租户就绪)
-- 见 docs/multi-tenant.md §10/§10.1。
--
-- ⚠️ 使用须知 (README 同步注明):
--   1. 本项目默认面向新装 (fresh-only), 本脚本为社区存量库的一次性例外,
--      执行前必须完成全库备份, 自担风险, 建议直接新装重建;
--   2. 执行前三步预处理 (§10.1):
--      ① 数据归属确认: 清理 A 类表中的测试/废弃行 (脏数据归入平台租户后
--         会永久污染平台视角的全租户混排列表);
--      ② uk 撞号预检 (有撞号先手工重命名加 -legacy2 后缀再跑本脚本):
--         SELECT username,  COUNT(*) c FROM sys_user  GROUP BY username  HAVING c>1;
--         SELECT code,      COUNT(*) c FROM sys_role   GROUP BY code      HAVING c>1;
--         SELECT post_code, COUNT(*) c FROM sys_post   GROUP BY post_code HAVING c>1;
--         SELECT config_key,COUNT(*) c FROM sys_config GROUP BY config_key HAVING c>1;
--      ③ 大表预案: MySQL 8.0 加列 INSTANT 秒级; uk 重建 INPLACE 仍需时间,
--         sys_audit_log/sys_job_log 百万级以上先归档裁剪或低峰执行 (更大规模 gh-ost/pt-osc);
--   3. 执行完成后: 开 config.yaml multiTenant.enable=true 并重启;
--      老登录 token 无 tid claim 自动回填平台租户 1, 不需要全员重登。
--
-- 幂等性: 加列/加索引前用 information_schema 存在性判断, 可重复执行。
-- ============================================================
SET NAMES utf8mb4 COLLATE utf8mb4_general_ci;
USE `hinay_admin`;

-- ------------------------------------------------------------
-- 0. 预检 (任一返回非空 → 先按须知 ② 处理撞号再继续)
-- ------------------------------------------------------------
-- (手工执行, 此处不设硬阻断)

-- ------------------------------------------------------------
-- 1. 租户表 + 应用授权表
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `sys_tenant` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name`          VARCHAR(128) NOT NULL              COMMENT '租户名称',
  `code`          VARCHAR(64)  NOT NULL              COMMENT '租户标识(全局唯一, 创建后不可变)',
  `contact`       VARCHAR(128) NOT NULL DEFAULT ''   COMMENT '联系人',
  `plan`          VARCHAR(32)  NOT NULL DEFAULT 'standard' COMMENT '套餐(预留)',
  `quota_users`   INT NOT NULL DEFAULT 0             COMMENT '用户数配额(0=不限, 预留)',
  `quota_storage` BIGINT NOT NULL DEFAULT 0          COMMENT '存储配额字节(0=不限, 预留)',
  `expire_at`     DATETIME DEFAULT NULL              COMMENT '到期时间(NULL=永久)',
  `status`        TINYINT NOT NULL DEFAULT 1         COMMENT '1=启用 0=禁用',
  `remark`        VARCHAR(255) NOT NULL DEFAULT ''   COMMENT '备注',
  `create_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `update_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at`    DATETIME DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='租户';

CREATE TABLE IF NOT EXISTS `sys_tenant_menu` (
  `tenant_id`  BIGINT UNSIGNED NOT NULL,
  `menu_id`    BIGINT UNSIGNED NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`tenant_id`, `menu_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='租户应用授权(目录/页面级, 按钮随父页面闭包)';

INSERT IGNORE INTO `sys_tenant` (`id`,`name`,`code`,`contact`,`status`,`remark`) VALUES
  (1, '平台租户', 'platform', '', 1, '内置平台租户(id=1), 不可禁用/删除');

INSERT IGNORE INTO `sys_tenant_menu` (`tenant_id`,`menu_id`) VALUES
  (1, 30), (1, 40), (1, 50), (1, 51), (1, 52),
  (1, 15), (1, 10), (1, 11), (1, 60), (1, 9040), (1, 9350);

-- ------------------------------------------------------------
-- 2. 租户管理员模板角色 (id=3) 与平台超管保护语义
-- ------------------------------------------------------------
INSERT IGNORE INTO `sys_role` (`id`,`name`,`code`,`sort`,`status`,`remark`,`data_scope`) VALUES
  (3, '租户管理员', 'tenant_admin', 3, 1, '租户管理员模板角色(创建租户时按域克隆, 域1快照即默认授权集)', 1);

-- ------------------------------------------------------------
-- 3. A 类表加 tenant_id 列 (INSTANT, 存量行默认归属平台租户 1)
--    通过存储过程按 information_schema 判断幂等加列/加索引/改 uk。
-- ------------------------------------------------------------
DROP PROCEDURE IF EXISTS `mt_upgrade`;
DELIMITER $$
CREATE PROCEDURE `mt_upgrade`()
BEGIN
  DECLARE db VARCHAR(64) DEFAULT DATABASE();

  -- 3.1 加 tenant_id 列 (存在即跳过)
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_user' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `sys_user`   ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(ORM按上下文注入)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_org' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `sys_org`    ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(ORM按上下文注入)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_role' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `sys_role`   ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(ORM按上下文注入)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_role_org' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `sys_role_org` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(ORM按上下文注入)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_post' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `sys_post`   ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(ORM按上下文注入)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_user_post' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `sys_user_post` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(ORM按上下文注入)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_config' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `sys_config` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(平台行=1; 白名单键支持租户覆盖)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_file' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `sys_file`   ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(ORM按上下文注入)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_audit_log' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `sys_audit_log` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(记录归属)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_login_log' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `sys_login_log` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(记录归属)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_job' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `sys_job`    ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(平台级任务=1)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_job_log' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `sys_job_log` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(记录归属)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='wf_definition' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `wf_definition` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(ORM按上下文注入)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='wf_instance' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `wf_instance` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(ORM按上下文注入)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='wf_task' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `wf_task`   ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(ORM按上下文注入)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='wf_record' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `wf_record` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(ORM按上下文注入)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='biz_leave' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `biz_leave` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(ORM按上下文注入)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='biz_message' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `biz_message` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(ORM按上下文注入)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='ai_conversation' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `ai_conversation` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(ORM按上下文注入)' AFTER `id`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='ai_chat_message' AND COLUMN_NAME='tenant_id') THEN
    ALTER TABLE `ai_chat_message` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '租户ID(跟随会话, ORM注入)' AFTER `id`;
  END IF;

  -- 3.2 唯一键租户化 (4 处) + 索引
  IF EXISTS (SELECT 1 FROM information_schema.STATISTICS
             WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_user' AND INDEX_NAME='uk_username') THEN
    ALTER TABLE `sys_user` DROP INDEX `uk_username`, ADD UNIQUE KEY `uk_tenant_username` (`tenant_id`,`username`);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS
             WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_user' AND INDEX_NAME='uk_tenant_username') THEN
    ALTER TABLE `sys_user` ADD UNIQUE KEY `uk_tenant_username` (`tenant_id`,`username`);
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.STATISTICS
             WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_role' AND INDEX_NAME='uk_code') THEN
    ALTER TABLE `sys_role` DROP INDEX `uk_code`, ADD UNIQUE KEY `uk_tenant_code` (`tenant_id`,`code`);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS
             WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_role' AND INDEX_NAME='uk_tenant_code') THEN
    ALTER TABLE `sys_role` ADD UNIQUE KEY `uk_tenant_code` (`tenant_id`,`code`);
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.STATISTICS
             WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_post' AND INDEX_NAME='uk_post_code') THEN
    ALTER TABLE `sys_post` DROP INDEX `uk_post_code`, ADD UNIQUE KEY `uk_tenant_post_code` (`tenant_id`,`post_code`);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS
             WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_post' AND INDEX_NAME='uk_tenant_post_code') THEN
    ALTER TABLE `sys_post` ADD UNIQUE KEY `uk_tenant_post_code` (`tenant_id`,`post_code`);
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.STATISTICS
             WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_config' AND INDEX_NAME='uk_config_key') THEN
    ALTER TABLE `sys_config` DROP INDEX `uk_config_key`, ADD UNIQUE KEY `uk_tenant_key` (`tenant_id`,`config_key`);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS
             WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_config' AND INDEX_NAME='uk_tenant_key') THEN
    ALTER TABLE `sys_config` ADD UNIQUE KEY `uk_tenant_key` (`tenant_id`,`config_key`);
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.STATISTICS
             WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_file' AND INDEX_NAME='idx_user') THEN
    ALTER TABLE `sys_file` DROP INDEX `idx_user`, ADD KEY `idx_tenant_user` (`tenant_id`,`user_id`);
  END IF;

  -- 3.3 idx_tenant (逐表 information_schema 判断, MySQL 8.0 无 ADD INDEX IF NOT EXISTS)
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_user'        AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `sys_user`        ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_org'         AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `sys_org`         ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_role'        AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `sys_role`        ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_role_org'    AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `sys_role_org`    ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_post'        AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `sys_post`        ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_user_post'   AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `sys_user_post`   ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_audit_log'   AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `sys_audit_log`   ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_login_log'   AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `sys_login_log`   ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_job'         AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `sys_job`         ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_job_log'     AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `sys_job_log`     ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='wf_definition'   AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `wf_definition`   ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='wf_instance'     AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `wf_instance`     ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='wf_task'         AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `wf_task`         ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='wf_record'       AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `wf_record`       ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='biz_leave'       AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `biz_leave`       ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='biz_message'     AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `biz_message`     ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='ai_conversation' AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `ai_conversation` ADD INDEX `idx_tenant` (`tenant_id`); END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=db AND TABLE_NAME='ai_chat_message' AND INDEX_NAME='idx_tenant') THEN ALTER TABLE `ai_chat_message` ADD INDEX `idx_tenant` (`tenant_id`); END IF;

  -- 3.4 sys_api 加 menu_id 归属列
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_api' AND COLUMN_NAME='menu_id') THEN
    ALTER TABLE `sys_api` ADD COLUMN `menu_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属菜单ID(0=平台/公共, 租户不可绑)' AFTER `description`;
  END IF;
END$$
DELIMITER ;

CALL `mt_upgrade`();
DROP PROCEDURE IF EXISTS `mt_upgrade`;

-- ------------------------------------------------------------
-- 4. casbin_rule 域化回填 (存量行全部归平台域 '1': p 行写 v3, g 行写 v2)
-- ------------------------------------------------------------
UPDATE `casbin_rule` SET `v3`='1' WHERE `ptype`='p' AND `v3`='';
UPDATE `casbin_rule` SET `v2`='1' WHERE `ptype`='g' AND `v2`='';

-- ------------------------------------------------------------
-- 5. tenant_admin 模板授权集 (域1快照 —— 基础协作包闭包, 与 init.sql 基线一致)
-- ------------------------------------------------------------
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (9300, 1, '租户管理', 2, '/system/tenants', 'system/tenants/index', 'OfficeBuilding', 'system:tenant:list', 18, 1, 1),
  (9301, 9300, '租户新增', 3, '', '', '', 'system:tenant:create', 1, 0, 1),
  (9302, 9300, '租户修改', 3, '', '', '', 'system:tenant:update', 2, 0, 1),
  (9303, 9300, '租户删除', 3, '', '', '', 'system:tenant:delete', 3, 0, 1),
  (9304, 9300, '应用授权', 3, '', '', '', 'system:tenant:assign', 4, 0, 1),
  (9305, 9300, '重置租户配置', 3, '', '', '', 'system:tenant:reset', 5, 0, 1),
  (9350, 1, '租户设置', 2, '/system/tenant-configs', 'system/tenant-configs/index', 'Setting', 'system:tenant-config:list', 19, 1, 1),
  (9351, 9350, '配置维护', 3, '', '', '', 'system:tenant-config:update', 1, 0, 1);

INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/tenants',                      'GET',    '租户管理', '租户分页列表'),
  ('/api/v1/system/tenants',                      'POST',   '租户管理', '创建租户(向导: 六件套事务)'),
  ('/api/v1/system/tenants/all',                  'GET',    '租户管理', '启用租户全量(平台切换器)'),
  ('/api/v1/system/tenants/{id}',                 'GET',    '租户管理', '租户详情'),
  ('/api/v1/system/tenants/{id}',                 'PUT',    '租户管理', '修改租户(code不可变)'),
  ('/api/v1/system/tenants/{id}',                 'DELETE', '租户管理', '删除租户(软删+code改写)'),
  ('/api/v1/system/tenants/{id}/menus',           'GET',    '租户管理', '获取租户应用授权集'),
  ('/api/v1/system/tenants/{id}/menus',           'PUT',    '租户管理', '设置应用授权集(双向联动casbin)'),
  ('/api/v1/system/tenants/{id}/configs/reset',   'POST',   '租户管理', '重置租户覆盖配置'),
  ('/api/v1/auth/tenant/info',                    'GET',    '认证',     '公开租户信息(按code, 仅显示名, 限流)');

-- sys_api.menu_id 归属回填 (同 init.sql 基线)
UPDATE `sys_api` SET `menu_id`=10  WHERE `group_name`='用户管理'  AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=11  WHERE `group_name`='角色管理'  AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=12  WHERE `group_name`='菜单管理'  AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=13  WHERE `group_name`='API管理'   AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=14  WHERE `group_name`='字典管理'  AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=15  WHERE `group_name`='文件管理'  AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=16  WHERE `group_name`='操作日志'  AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=17  WHERE `group_name`='全局配置'  AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=18  WHERE `group_name`='登录日志'  AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=19  WHERE `group_name`='在线用户'  AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=20  WHERE `group_name`='定时任务'  AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=21  WHERE `group_name`='代码生成'  AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=22  WHERE `group_name`='服务器监控' AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=60  WHERE `group_name`='组织机构'  AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=51 WHERE `group_name`='消息通知' AND `menu_id`=0
  AND `path` IN ('/api/v1/message','/api/v1/message/system','/api/v1/message/:id');
UPDATE `sys_api` SET `menu_id`=52 WHERE `group_name`='消息通知' AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=9000 WHERE `group_name`='审批中心' AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=9040 WHERE `group_name`='岗位管理' AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=9100 WHERE `group_name`='AI助手'   AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=9200 WHERE `group_name`='业务审批' AND `menu_id`=0;
UPDATE `sys_api` SET `menu_id`=9300 WHERE `group_name`='租户管理' AND `menu_id`=0;

-- tenant_admin 模板角色 (id=3, 域1) 菜单策略
INSERT IGNORE INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', '3', 'menu:30', 'access', '1', '', ''),
  ('p', '3', 'menu:40', 'access', '1', '', ''),
  ('p', '3', 'menu:50', 'access', '1', '', ''),
  ('p', '3', 'menu:51', 'access', '1', '', ''),
  ('p', '3', 'menu:52', 'access', '1', '', ''),
  ('p', '3', 'menu:511', 'access', '1', '', ''),
  ('p', '3', 'menu:512', 'access', '1', '', ''),
  ('p', '3', 'menu:521', 'access', '1', '', ''),
  ('p', '3', 'menu:522', 'access', '1', '', ''),
  ('p', '3', 'menu:15', 'access', '1', '', ''),
  ('p', '3', 'menu:151', 'access', '1', '', ''),
  ('p', '3', 'menu:152', 'access', '1', '', ''),
  ('p', '3', 'menu:10', 'access', '1', '', ''),
  ('p', '3', 'menu:101', 'access', '1', '', ''),
  ('p', '3', 'menu:102', 'access', '1', '', ''),
  ('p', '3', 'menu:103', 'access', '1', '', ''),
  ('p', '3', 'menu:11', 'access', '1', '', ''),
  ('p', '3', 'menu:111', 'access', '1', '', ''),
  ('p', '3', 'menu:112', 'access', '1', '', ''),
  ('p', '3', 'menu:113', 'access', '1', '', ''),
  ('p', '3', 'menu:114', 'access', '1', '', ''),
  ('p', '3', 'menu:60', 'access', '1', '', ''),
  ('p', '3', 'menu:601', 'access', '1', '', ''),
  ('p', '3', 'menu:602', 'access', '1', '', ''),
  ('p', '3', 'menu:603', 'access', '1', '', ''),
  ('p', '3', 'menu:9040', 'access', '1', '', ''),
  ('p', '3', 'menu:9350', 'access', '1', '', ''),
  ('p', '3', 'menu:9351', 'access', '1', '', ''),
  ('p', '3', 'menu:9041', 'access', '1', '', ''),
  ('p', '3', 'menu:9042', 'access', '1', '', ''),
  ('p', '3', 'menu:9043', 'access', '1', '', '');

-- tenant_admin 模板角色 (id=3, 域1) API 策略
INSERT IGNORE INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', '3', '/api/v1/system/users',              'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/users',              'POST',   '1', '', ''),
  ('p', '3', '/api/v1/system/users/:id',          'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/users/:id',          'PUT',    '1', '', ''),
  ('p', '3', '/api/v1/system/users/:id',          'DELETE', '1', '', ''),
  ('p', '3', '/api/v1/system/users/:id/roles',    'PUT',    '1', '', ''),
  ('p', '3', '/api/v1/system/users/{id}/password','PUT',    '1', '', ''),
  ('p', '3', '/api/v1/system/users/export',       'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/users/import',       'POST',   '1', '', ''),
  ('p', '3', '/api/v1/system/users/import-template', 'GET', '1', '', ''),
  ('p', '3', '/api/v1/system/roles',              'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/roles',              'POST',   '1', '', ''),
  ('p', '3', '/api/v1/system/roles/all',          'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/roles/:id',          'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/roles/:id',          'PUT',    '1', '', ''),
  ('p', '3', '/api/v1/system/roles/:id',          'DELETE', '1', '', ''),
  ('p', '3', '/api/v1/system/roles/:id/menus',    'PUT',    '1', '', ''),
  ('p', '3', '/api/v1/system/roles/:id/menus',    'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/roles/:id/apis',     'PUT',    '1', '', ''),
  ('p', '3', '/api/v1/system/roles/:id/apis',     'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/menus/tree',        'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/apis/all',          'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/configs',           'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/configs',           'POST',   '1', '', ''),
  ('p', '3', '/api/v1/system/configs/:id',       'PUT',    '1', '', ''),
  ('p', '3', '/api/v1/system/orgs',               'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/orgs',               'POST',   '1', '', ''),
  ('p', '3', '/api/v1/system/orgs/:id',           'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/orgs/:id',           'PUT',    '1', '', ''),
  ('p', '3', '/api/v1/system/orgs/:id',           'DELETE', '1', '', ''),
  ('p', '3', '/api/v1/system/orgs/tree',          'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/posts',              'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/posts/all',          'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/posts',              'POST',   '1', '', ''),
  ('p', '3', '/api/v1/system/posts/{id}',         'PUT',    '1', '', ''),
  ('p', '3', '/api/v1/system/posts/{id}',         'DELETE', '1', '', ''),
  ('p', '3', '/api/v1/system/posts/{id}/members', 'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/posts/{id}/members', 'POST',   '1', '', ''),
  ('p', '3', '/api/v1/system/posts/members/{relId}', 'DELETE', '1', '', ''),
  ('p', '3', '/api/v1/system/files',              'GET',    '1', '', ''),
  ('p', '3', '/api/v1/system/files/upload',       'POST',   '1', '', ''),
  ('p', '3', '/api/v1/system/files/presign',      'POST',   '1', '', ''),
  ('p', '3', '/api/v1/system/files/presign/confirm', 'POST', '1', '', ''),
  ('p', '3', '/api/v1/system/files/:id',          'DELETE', '1', '', ''),
  ('p', '3', '/api/v1/message',                   'GET',    '1', '', ''),
  ('p', '3', '/api/v1/message/system',            'POST',   '1', '', ''),
  ('p', '3', '/api/v1/message/:id',               'GET',    '1', '', ''),
  ('p', '3', '/api/v1/message/:id',               'DELETE', '1', '', ''),
  ('p', '3', '/api/v1/message/inbox',             'GET',    '1', '', ''),
  ('p', '3', '/api/v1/message/inbox/:id',         'GET',    '1', '', ''),
  ('p', '3', '/api/v1/message/inbox/:id',         'DELETE', '1', '', ''),
  ('p', '3', '/api/v1/message/unread-count',      'GET',    '1', '', ''),
  ('p', '3', '/api/v1/message/read-all',          'PUT',    '1', '', ''),
  ('p', '3', '/api/v1/message/*/read',            'PUT',    '1', '', ''),
  ('p', '3', '/api/v1/message/private',           'POST',   '1', '', '');

-- ------------------------------------------------------------
-- 6. 存量数据归属平台租户 (加列默认 1 已覆盖; 此处显式兜底 NULL/0 异常行)
-- ------------------------------------------------------------
UPDATE `sys_user`   SET `tenant_id`=1 WHERE `tenant_id`=0;
UPDATE `sys_org`    SET `tenant_id`=1 WHERE `tenant_id`=0;
UPDATE `sys_role`   SET `tenant_id`=1 WHERE `tenant_id`=0;
UPDATE `sys_config` SET `tenant_id`=1 WHERE `tenant_id`=0;

-- ------------------------------------------------------------
-- 7. 租户套餐 (§7.2 应用授权集模板复用层; 已是最新多租户基线的库仅需执行本段)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `sys_tenant_package` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name`       VARCHAR(64)  NOT NULL                COMMENT '套餐名称(全局唯一, 软删时改写防撞uk)',
  `menu_ids`   TEXT         NOT NULL                COMMENT '应用授权集模板(目录/页面级ID的JSON数组)',
  `status`     TINYINT      NOT NULL DEFAULT 1      COMMENT '1=启用 0=停用',
  `remark`     VARCHAR(255) NOT NULL DEFAULT ''     COMMENT '备注',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='租户套餐(应用授权集模板)';

DROP PROCEDURE IF EXISTS `mt_pkg_upgrade`;
DELIMITER $$
CREATE PROCEDURE `mt_pkg_upgrade`()
BEGIN
  DECLARE db VARCHAR(64);
  SELECT DATABASE() INTO db;
  IF NOT EXISTS (SELECT 1 FROM information_schema.COLUMNS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_tenant' AND COLUMN_NAME='package_id') THEN
    ALTER TABLE `sys_tenant` ADD COLUMN `package_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '套餐ID(sys_tenant_package, 0=自定义/未绑定)' AFTER `plan`;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.STATISTICS
                 WHERE TABLE_SCHEMA=db AND TABLE_NAME='sys_tenant' AND INDEX_NAME='idx_package') THEN
    ALTER TABLE `sys_tenant` ADD INDEX `idx_package` (`package_id`);
  END IF;
END$$
DELIMITER ;

CALL `mt_pkg_upgrade`();
DROP PROCEDURE IF EXISTS `mt_pkg_upgrade`;

INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (9360, 1, '租户套餐', 2, '/system/tenant-packages', 'system/tenant-packages/index', 'Box', 'system:tenant-package:list', 20, 1, 1),
  (9361, 9360, '套餐新增', 3, '', '', '', 'system:tenant-package:create', 1, 0, 1),
  (9362, 9360, '套餐修改', 3, '', '', '', 'system:tenant-package:update', 2, 0, 1),
  (9363, 9360, '套餐删除', 3, '', '', '', 'system:tenant-package:delete', 3, 0, 1),
  (9364, 9360, '套餐同步', 3, '', '', '', 'system:tenant-package:sync',   4, 0, 1);

INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/tenant-packages',           'GET',    '租户套餐', '套餐分页列表'),
  ('/api/v1/system/tenant-packages/all',       'GET',    '租户套餐', '启用套餐全量(创建向导/授权抽屉下拉)'),
  ('/api/v1/system/tenant-packages/{id}',      'GET',    '租户套餐', '套餐详情(含授权集模板)'),
  ('/api/v1/system/tenant-packages',           'POST',   '租户套餐', '新增套餐'),
  ('/api/v1/system/tenant-packages/{id}',      'PUT',    '租户套餐', '修改套餐'),
  ('/api/v1/system/tenant-packages/{id}',      'DELETE', '租户套餐', '删除套餐(使用中禁止)'),
  ('/api/v1/system/tenant-packages/{id}/sync', 'POST',   '租户套餐', '套餐同步至绑定租户(重写授权集+联动casbin)');

UPDATE `sys_api` SET `menu_id`=9360 WHERE `group_name`='租户套餐' AND `menu_id`=0;

-- 默认套餐种子 (与 §5 tenant_admin 模板授权集 / init.sql 基线一致)
INSERT IGNORE INTO `sys_tenant_package` (`id`,`name`,`menu_ids`,`status`,`remark`) VALUES
  (1, '基础协作包', '[30,40,50,51,52,15,10,11,60,9040,9350]', 1, '默认套餐: 仪表盘/个人中心/消息/文件/用户/角色/组织/岗位/租户设置');

-- ------------------------------------------------------------
-- 完成后: ① config.yaml 增加 multiTenant.enable: true; ② 重启应用。
-- ------------------------------------------------------------

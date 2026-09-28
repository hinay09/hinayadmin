-- ============================================================
-- 审计字段 create_id / update_id (增量升级脚本: 适用于已按旧版 init.sql 初始化的存量库)
-- 全新安装直接执行 init.sql, 无需本文件。
--
-- 说明: 由 utility/ormfill 通过 ORM 钩子自动填充 (INSERT 补 create_id+update_id,
-- UPDATE 补 update_id, 取当前登录用户; 无登录上下文的后台写入保持 0)。
-- casbin_rule 与三张日志表 (sys_audit_log / sys_login_log / sys_job_log)
-- 及两张无 created_at 的消息关联表不纳入审计字段。
-- 执行方式: mysql -u<user> -p <库名> < manifest/sql/upgrade/0007_create_update_id.sql
-- ============================================================

ALTER TABLE `sys_user`
  ADD COLUMN `create_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID(ORM自动填充)',
  ADD COLUMN `update_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后修改人ID(ORM自动填充)';

ALTER TABLE `sys_role`
  ADD COLUMN `create_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID(ORM自动填充)',
  ADD COLUMN `update_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后修改人ID(ORM自动填充)';

ALTER TABLE `sys_role_org`
  ADD COLUMN `create_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID(ORM自动填充)',
  ADD COLUMN `update_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后修改人ID(ORM自动填充)';

ALTER TABLE `sys_org`
  ADD COLUMN `create_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID(ORM自动填充)',
  ADD COLUMN `update_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后修改人ID(ORM自动填充)';

ALTER TABLE `sys_menu`
  ADD COLUMN `create_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID(ORM自动填充)',
  ADD COLUMN `update_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后修改人ID(ORM自动填充)';

ALTER TABLE `sys_api`
  ADD COLUMN `create_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID(ORM自动填充)',
  ADD COLUMN `update_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后修改人ID(ORM自动填充)';

ALTER TABLE `sys_dict_type`
  ADD COLUMN `create_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID(ORM自动填充)',
  ADD COLUMN `update_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后修改人ID(ORM自动填充)';

ALTER TABLE `sys_dict_data`
  ADD COLUMN `create_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID(ORM自动填充)',
  ADD COLUMN `update_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后修改人ID(ORM自动填充)';

ALTER TABLE `sys_config`
  ADD COLUMN `create_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID(ORM自动填充)',
  ADD COLUMN `update_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后修改人ID(ORM自动填充)';

ALTER TABLE `sys_file`
  ADD COLUMN `create_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID(ORM自动填充)',
  ADD COLUMN `update_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后修改人ID(ORM自动填充)';

ALTER TABLE `sys_job`
  ADD COLUMN `create_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID(ORM自动填充)',
  ADD COLUMN `update_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后修改人ID(ORM自动填充)';

ALTER TABLE `biz_message`
  ADD COLUMN `create_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建人ID(ORM自动填充)',
  ADD COLUMN `update_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后修改人ID(ORM自动填充)';

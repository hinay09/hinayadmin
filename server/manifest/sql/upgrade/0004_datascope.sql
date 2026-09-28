-- ============================================================
-- 组织数据权限 (增量升级脚本: 适用于已按旧版 init.sql 初始化的存量库)
-- 全新安装直接执行 init.sql, 无需本文件。
-- 执行方式: mysql -u<user> -p <库名> < manifest/sql/upgrade/0004_datascope.sql
-- ============================================================

-- 1. sys_role 追加数据范围字段 (默认 1=全部, 保持存量角色行为不变)
ALTER TABLE `sys_role`
  ADD COLUMN `data_scope` TINYINT NOT NULL DEFAULT 1
    COMMENT '数据范围:1=全部,2=自定义,3=本部门,4=本部门及以下,5=仅本人' AFTER `remark`;

-- 2. 角色 <-> 自定义数据范围组织 绑定表
CREATE TABLE IF NOT EXISTS `sys_role_org` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `role_id`    BIGINT UNSIGNED NOT NULL                COMMENT '角色ID',
  `org_id`     BIGINT UNSIGNED NOT NULL                COMMENT '组织ID',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_org` (`role_id`, `org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色自定义数据范围组织绑定';

-- 存量数据无需迁移: 默认 data_scope=1 与升级前"不过滤"的行为一致。

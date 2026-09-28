-- ============================================================
-- 登录日志 (增量升级脚本: 适用于已按旧版 init.sql 初始化的存量库)
-- 全新安装直接执行 init.sql, 无需本文件。
-- 执行方式: mysql -u<user> -p <库名> < manifest/sql/upgrade/0001_login_log.sql
-- ============================================================

-- 1. sys_user 追加最近登录字段
ALTER TABLE `sys_user`
  ADD COLUMN `last_login_at` DATETIME     DEFAULT NULL           COMMENT '最近登录时间' AFTER `remark`,
  ADD COLUMN `last_login_ip` VARCHAR(64)  NOT NULL DEFAULT ''    COMMENT '最近登录IP' AFTER `last_login_at`;

-- 2. 登录日志表
CREATE TABLE IF NOT EXISTS `sys_login_log` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `user_id`    BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '用户ID(登录用户不存在时为0)',
  `username`   VARCHAR(64)  NOT NULL DEFAULT ''        COMMENT '登录账号',
  `status`     TINYINT      NOT NULL DEFAULT 0         COMMENT '结果:1=成功,0=失败',
  `message`    VARCHAR(255) NOT NULL DEFAULT ''        COMMENT '失败原因(成功时为空)',
  `ip`         VARCHAR(64)  NOT NULL DEFAULT ''        COMMENT '登录IP',
  `user_agent` VARCHAR(512) NOT NULL DEFAULT ''        COMMENT 'User-Agent',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_user` (`user_id`),
  KEY `idx_username` (`username`),
  KEY `idx_status` (`status`),
  KEY `idx_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='登录日志';

-- 3. 菜单: 登录日志 (顶级, 与操作日志并列)
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (18, 0, '登录日志', 2, '/system/login-logs', 'system/login-logs/index', 'Key', 'system:login-log:list', 100, 1, 1);

-- 按钮权限
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (181, 18, '登录日志删除', 3, '', '', '', 'system:login-log:delete', 1, 0, 1),
  (182, 18, '登录日志清空', 3, '', '', '', 'system:login-log:clear', 2, 0, 1);

-- 4. API 资源
INSERT INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/login-logs',      'GET',    '登录日志', '登录日志列表'),
  ('/api/v1/system/login-logs/:id',  'DELETE', '登录日志', '删除登录日志'),
  ('/api/v1/system/login-logs',      'DELETE', '登录日志', '清空登录日志');

-- 5. Casbin: admin 角色
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', 'admin', 'menu:18',  'access', '', '', ''),
  ('p', 'admin', 'menu:181', 'access', '', '', ''),
  ('p', 'admin', 'menu:182', 'access', '', '', '');

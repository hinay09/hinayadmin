-- ============================================================
-- 密码策略 (增量升级脚本: 适用于已按旧版 init.sql 初始化的存量库)
-- 全新安装直接执行 init.sql, 无需本文件。
-- 执行方式: mysql -u<user> -p <库名> < manifest/sql/upgrade/0008_password_policy.sql
-- ============================================================

-- 1. sys_user 追加密码策略字段
ALTER TABLE `sys_user`
  ADD COLUMN `pwd_updated_at` DATETIME  DEFAULT NULL           COMMENT '密码最后修改时间(有效期计算)' AFTER `last_login_ip`,
  ADD COLUMN `must_change_pwd` TINYINT  NOT NULL DEFAULT 0     COMMENT '强制改密:1=下次登录须改密'    AFTER `pwd_updated_at`;

-- 2. 密码策略配置项 (全局配置页可改, 实时生效)
INSERT INTO `sys_config` (`config_key`, `config_value`, `config_type`, `name`, `remark`, `sort`) VALUES
  ('sys.password.min_length',      '6',  1, '密码最小长度', '设置新密码时的最小长度', 10),
  ('sys.password.max_length',      '32', 1, '密码最大长度', '设置新密码时的最大长度', 11),
  ('sys.password.require_upper',   'false', 2, '密码需含大写字母', '设置新密码时校验', 12),
  ('sys.password.require_lower',   'false', 2, '密码需含小写字母', '设置新密码时校验', 13),
  ('sys.password.require_digit',   'false', 2, '密码需含数字',     '设置新密码时校验', 14),
  ('sys.password.require_special', 'false', 2, '密码需含特殊字符', '设置新密码时校验', 15),
  ('sys.password.expire_days',     '0',  1, '密码有效期(天)', '0=永不过期; 过期后登录强制改密', 16);

-- 3. 存量库的内置 admin 使用公开默认密码, 升级后首次登录强制改密
--    (其余存量用户 pwd_updated_at 为 NULL: 仅在开启有效期后才被视为需改密)
UPDATE `sys_user` SET `must_change_pwd` = 1 WHERE `username` = 'admin';

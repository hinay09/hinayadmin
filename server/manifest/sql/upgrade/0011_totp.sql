-- ============================================================
-- 0011: TOTP 两步验证 (2FA)
-- 个人中心自助绑定/解绑; 登录密码通过后需二次输入动态验证码。
-- 说明: 本表无 create_id/update_id 审计列(行为人即 user_id 本身),
--       不纳入 ormfill fillTables 白名单;
--       解绑走物理删除(Unscoped), 避免 uk_user_id 与软删残留行冲突。
-- ============================================================

CREATE TABLE IF NOT EXISTS `sys_user_totp` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id`    BIGINT UNSIGNED NOT NULL                COMMENT '用户ID(唯一)',
  `secret`     VARCHAR(64)  NOT NULL                   COMMENT 'TOTP 密钥(Base32)',
  `enabled`    TINYINT      NOT NULL DEFAULT 0         COMMENT '状态:0=待验证(已生成未绑定),1=已启用',
  `last_step`  BIGINT       NOT NULL DEFAULT 0         COMMENT '最近已消费的时间步(Unix/30, 防验证码重放)',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME     DEFAULT NULL               COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户TOTP两步验证';

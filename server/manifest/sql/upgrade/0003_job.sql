-- ============================================================
-- 定时任务 (增量升级脚本: 适用于已按旧版 init.sql 初始化的存量库)
-- 全新安装直接执行 init.sql, 无需本文件。
-- 执行方式: mysql -u<user> -p <库名> < manifest/sql/upgrade/0003_job.sql
-- ============================================================

CREATE TABLE IF NOT EXISTS `sys_job` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '任务ID',
  `name`       VARCHAR(64)  NOT NULL                COMMENT '任务名称',
  `handler`    VARCHAR(128) NOT NULL                COMMENT '处理器名称(需已注册)',
  `cron_expr`  VARCHAR(64)  NOT NULL                COMMENT 'cron表达式(6位: 秒 分 时 日 月 周)',
  `params`     VARCHAR(512) NOT NULL DEFAULT ''     COMMENT '任务参数(JSON, 可空)',
  `status`     TINYINT      NOT NULL DEFAULT 0      COMMENT '状态:1=启动,0=暂停',
  `remark`     VARCHAR(255) NOT NULL DEFAULT ''     COMMENT '备注',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME     DEFAULT NULL            COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='定时任务';

CREATE TABLE IF NOT EXISTS `sys_job_log` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '日志ID',
  `job_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '任务ID',
  `job_name`    VARCHAR(64)  NOT NULL DEFAULT ''        COMMENT '任务名称(冗余, 删除任务后日志仍可读)',
  `handler`     VARCHAR(128) NOT NULL DEFAULT ''        COMMENT '处理器名称',
  `params`      VARCHAR(512) NOT NULL DEFAULT ''        COMMENT '任务参数(JSON)',
  `status`      TINYINT      NOT NULL DEFAULT 0         COMMENT '结果:1=成功,0=失败',
  `output`      VARCHAR(1024) NOT NULL DEFAULT ''       COMMENT '执行输出/失败原因',
  `duration_ms` INT          NOT NULL DEFAULT 0         COMMENT '耗时(毫秒)',
  `created_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_job` (`job_id`),
  KEY `idx_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='定时任务执行日志';

-- 菜单: 定时任务 (顶级)
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (20, 0, '定时任务', 2, '/system/jobs', 'system/jobs/index', 'AlarmClock', 'system:job:list', 102, 1, 1);

-- 按钮权限
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (201, 20, '任务新增', 3, '', '', '', 'system:job:create', 1, 0, 1),
  (202, 20, '任务修改', 3, '', '', '', 'system:job:update', 2, 0, 1),
  (203, 20, '任务删除', 3, '', '', '', 'system:job:delete', 3, 0, 1),
  (204, 20, '任务启停', 3, '', '', '', 'system:job:status', 4, 0, 1),
  (205, 20, '立即执行', 3, '', '', '', 'system:job:run', 5, 0, 1),
  (206, 20, '执行日志', 3, '', '', '', 'system:job:log', 6, 0, 1);

-- API 资源
INSERT INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/jobs',              'GET',    '定时任务', '任务列表'),
  ('/api/v1/system/jobs',              'POST',   '定时任务', '新增任务'),
  ('/api/v1/system/jobs/:id',          'PUT',    '定时任务', '修改任务'),
  ('/api/v1/system/jobs/:id',          'DELETE', '定时任务', '删除任务'),
  ('/api/v1/system/jobs/:id/status',   'PUT',    '定时任务', '启动/暂停任务'),
  ('/api/v1/system/jobs/:id/run',      'POST',   '定时任务', '立即执行一次'),
  ('/api/v1/system/jobs/logs',         'GET',    '定时任务', '执行日志列表'),
  ('/api/v1/system/jobs/handlers',     'GET',    '定时任务', '已注册处理器列表');

-- Casbin: admin 角色
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', 'admin', 'menu:20',  'access', '', '', ''),
  ('p', 'admin', 'menu:201', 'access', '', '', ''),
  ('p', 'admin', 'menu:202', 'access', '', '', ''),
  ('p', 'admin', 'menu:203', 'access', '', '', ''),
  ('p', 'admin', 'menu:204', 'access', '', '', ''),
  ('p', 'admin', 'menu:205', 'access', '', '', ''),
  ('p', 'admin', 'menu:206', 'access', '', '', '');

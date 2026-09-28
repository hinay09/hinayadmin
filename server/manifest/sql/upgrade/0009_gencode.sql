-- ============================================================
-- 网页版代码生成 (增量升级脚本)
-- 全新安装直接执行 init.sql, 无需本文件。
-- 执行方式: mysql -u<user> -p <库名> < manifest/sql/upgrade/0009_gencode.sql
-- 注意: 接口受配置 gencode.enable 控制 (默认关闭), 开发环境在 config.yaml 开启。
-- ============================================================

INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (21, 0, '代码生成', 2, '/system/gencode', 'system/gencode/index', 'MagicStick', 'system:gencode:list', 103, 1, 1);

INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (211, 21, '生成预览', 3, '', '', '', 'system:gencode:preview', 1, 0, 1),
  (212, 21, '打包下载', 3, '', '', '', 'system:gencode:download', 2, 0, 1),
  (213, 21, '写入源码', 3, '', '', '', 'system:gencode:write', 3, 0, 1);

INSERT INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/gencode/tables',    'GET',  '代码生成', '可生成表清单'),
  ('/api/v1/system/gencode/columns',   'GET',  '代码生成', '表列信息'),
  ('/api/v1/system/gencode/preview',   'POST', '代码生成', '预览生成代码'),
  ('/api/v1/system/gencode/download',  'GET',  '代码生成', '下载生成代码(zip)'),
  ('/api/v1/system/gencode/write',     'POST', '代码生成', '生成并写入源码树');

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', 'admin', 'menu:21',  'access', '', '', ''),
  ('p', 'admin', 'menu:211', 'access', '', '', ''),
  ('p', 'admin', 'menu:212', 'access', '', '', ''),
  ('p', 'admin', 'menu:213', 'access', '', '', '');

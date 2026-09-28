-- ============================================================
-- Excel 导入导出 (增量升级脚本: 用户模块示例 API 资源)
-- 全新安装直接执行 init.sql, 无需本文件。
-- 导出按钮复用 system:user:list 权限, 导入复用 system:user:create 权限, 无需新增菜单。
-- 执行方式: mysql -u<user> -p <库名> < manifest/sql/upgrade/0005_excel.sql
-- ============================================================

INSERT INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/users/export',           'GET',  '用户管理', '用户列表导出'),
  ('/api/v1/system/users/import',           'POST', '用户管理', '用户导入'),
  ('/api/v1/system/users/import-template',  'GET',  '用户管理', '用户导入模板下载');

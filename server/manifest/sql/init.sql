-- ============================================================
-- Hinay Admin 数据库初始化脚本
-- 数据库: hinay_admin (MySQL 8+, utf8mb4)
-- ============================================================

-- 强制设置当前会话字符集为 utf8mb4，防止因 Docker 容器 locale
-- 导致 mysql 客户端默认 latin1 而引发中文乱码
SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE DATABASE IF NOT EXISTS `hinay_admin`
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;

USE `hinay_admin`;

-- ------------------------------------------------------------
-- 用户表
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `sys_user`;
CREATE TABLE `sys_user` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户ID',
  `username`   VARCHAR(64)  NOT NULL                COMMENT '登录账号',
  `password`   VARCHAR(128) NOT NULL                COMMENT 'bcrypt 加密密码',
  `nickname`   VARCHAR(64)  NOT NULL DEFAULT ''     COMMENT '昵称',
  `avatar`     VARCHAR(255) NOT NULL DEFAULT ''     COMMENT '头像URL',
  `email`      VARCHAR(128) NOT NULL DEFAULT ''     COMMENT '邮箱',
  `phone`      VARCHAR(32)  NOT NULL DEFAULT ''     COMMENT '手机号',
  `org_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '所属组织ID',
  `status`     TINYINT      NOT NULL DEFAULT 1      COMMENT '状态:1=启用,0=禁用',
  `remark`     VARCHAR(255) NOT NULL DEFAULT ''     COMMENT '备注',
  `last_login_at` DATETIME  DEFAULT NULL              COMMENT '最近登录时间',
  `last_login_ip` VARCHAR(64) NOT NULL DEFAULT ''    COMMENT '最近登录IP',
  `pwd_updated_at` DATETIME  DEFAULT NULL              COMMENT '密码最后修改时间(有效期计算)',
  `must_change_pwd` TINYINT  NOT NULL DEFAULT 0      COMMENT '强制改密:1=下次登录须改密',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '最后修改人ID(ORM自动填充)',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME     DEFAULT NULL            COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统用户';

-- ------------------------------------------------------------
-- 组织机构表(无限级树形)
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `sys_org`;
CREATE TABLE `sys_org` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '组织ID',
  `parent_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0     COMMENT '父级ID, 0=顶级',
  `name`       VARCHAR(64)  NOT NULL                COMMENT '组织名称',
  `leader`     VARCHAR(64)  NOT NULL DEFAULT ''     COMMENT '负责人',
  `phone`      VARCHAR(32)  NOT NULL DEFAULT ''     COMMENT '联系电话',
  `email`      VARCHAR(128) NOT NULL DEFAULT ''     COMMENT '邮箱',
  `sort`       INT          NOT NULL DEFAULT 0      COMMENT '排序',
  `status`     TINYINT      NOT NULL DEFAULT 1      COMMENT '状态:1=启用,0=禁用',
  `remark`     VARCHAR(255) NOT NULL DEFAULT ''     COMMENT '备注',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '最后修改人ID(ORM自动填充)',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME     DEFAULT NULL            COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  KEY `idx_parent` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='组织机构';

-- ------------------------------------------------------------
-- 角色表
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `sys_role`;
CREATE TABLE `sys_role` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '角色ID',
  `name`       VARCHAR(64)  NOT NULL                COMMENT '角色名称',
  `code`       VARCHAR(64)  NOT NULL                COMMENT '角色编码',
  `sort`       INT          NOT NULL DEFAULT 0      COMMENT '排序',
  `status`     TINYINT      NOT NULL DEFAULT 1      COMMENT '状态:1=启用,0=禁用',
  `remark`     VARCHAR(255) NOT NULL DEFAULT ''     COMMENT '备注',
  `data_scope` TINYINT      NOT NULL DEFAULT 1      COMMENT '数据范围:1=全部,2=自定义,3=本部门,4=本部门及以下,5=仅本人',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '最后修改人ID(ORM自动填充)',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME     DEFAULT NULL            COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统角色';

-- ------------------------------------------------------------
-- 角色自定义数据范围 <-> 组织 绑定表
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `sys_role_org`;
CREATE TABLE `sys_role_org` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `role_id`    BIGINT UNSIGNED NOT NULL                COMMENT '角色ID',
  `org_id`     BIGINT UNSIGNED NOT NULL                COMMENT '组织ID',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '最后修改人ID(ORM自动填充)',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_org` (`role_id`, `org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色自定义数据范围组织绑定';

-- ------------------------------------------------------------
-- 菜单表(目录/菜单/按钮)
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `sys_menu`;
CREATE TABLE `sys_menu` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '菜单ID',
  `parent_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '父级ID',
  `name`       VARCHAR(64)  NOT NULL                COMMENT '菜单名称',
  `type`       TINYINT      NOT NULL DEFAULT 2      COMMENT '类型:1=目录,2=菜单,3=按钮',
  `path`       VARCHAR(128) NOT NULL DEFAULT ''     COMMENT '路由路径',
  `component`  VARCHAR(128) NOT NULL DEFAULT ''     COMMENT '组件路径',
  `icon`       VARCHAR(64)  NOT NULL DEFAULT ''     COMMENT '图标',
  `permission` VARCHAR(128) NOT NULL DEFAULT ''     COMMENT '权限标识',
  `sort`       INT          NOT NULL DEFAULT 0      COMMENT '排序',
  `visible`    TINYINT      NOT NULL DEFAULT 1      COMMENT '是否显示:1=是,0=否',
  `status`     TINYINT      NOT NULL DEFAULT 1      COMMENT '状态:1=启用,0=禁用',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '最后修改人ID(ORM自动填充)',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME     DEFAULT NULL            COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  KEY `idx_parent` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统菜单';

-- ------------------------------------------------------------
-- API接口资源表
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `sys_api`;
CREATE TABLE `sys_api` (
  `id`          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `path`        VARCHAR(255) NOT NULL COMMENT 'API路径',
  `method`      VARCHAR(10)  NOT NULL COMMENT 'HTTP方法(GET/POST/PUT/DELETE)',
  `group_name`  VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '分组名称',
  `description` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '接口描述',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '最后修改人ID(ORM自动填充)',
  `created_at`  DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at`  DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at`  DATETIME DEFAULT NULL COMMENT '删除时间(软删)',
  UNIQUE KEY `uk_path_method` (`path`, `method`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='API接口资源表';

-- ------------------------------------------------------------
-- Casbin 策略表 (兼容标准 casbin gorm-adapter 字段布局)
-- 字段语义:
--   ptype = p  (策略)        : v0=sub(role), v1=obj(path), v2=act(method), v3~v5 预留
--   ptype = g  (角色继承/分组): v0=user/sub, v1=role,        v2=domain (可选)
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `casbin_rule`;
CREATE TABLE `casbin_rule` (
  `id`    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT             COMMENT '主键',
  `ptype` VARCHAR(100) NOT NULL DEFAULT ''                    COMMENT '策略类型: p / g / g2 ...',
  `v0`    VARCHAR(100) NOT NULL DEFAULT ''                    COMMENT 'p:sub(role) | g:user',
  `v1`    VARCHAR(100) NOT NULL DEFAULT ''                    COMMENT 'p:obj(path) | g:role',
  `v2`    VARCHAR(100) NOT NULL DEFAULT ''                    COMMENT 'p:act(method) | g:domain',
  `v3`    VARCHAR(100) NOT NULL DEFAULT ''                    COMMENT '预留字段',
  `v4`    VARCHAR(100) NOT NULL DEFAULT ''                    COMMENT '预留字段',
  `v5`    VARCHAR(100) NOT NULL DEFAULT ''                    COMMENT '预留字段',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`),
  KEY `idx_ptype` (`ptype`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Casbin 策略';

-- ------------------------------------------------------------
-- 消息通知: 主表 (系统通知 + 私信共用)
--   type=1 系统通知 (target_scope: 1=全员 / 2=指定角色 / 3=指定用户)
--   type=2 私信通知 (receiver_id 为接收人, target_scope=0)
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `biz_message`;
CREATE TABLE `biz_message` (
  `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `type`         TINYINT      NOT NULL                COMMENT '消息类型:1=系统通知,2=私信',
  `title`        VARCHAR(128) NOT NULL                COMMENT '标题',
  `content`      TEXT         NOT NULL                COMMENT '内容',
  `level`        TINYINT      NOT NULL DEFAULT 1      COMMENT '级别:1=普通,2=重要,3=紧急',
  `sender_id`    BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '发送人ID',
  `target_scope` TINYINT      NOT NULL DEFAULT 0      COMMENT '系统通知范围:1=all,2=role,3=user;私信=0',
  `receiver_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '私信接收者ID',
  `status`       TINYINT      NOT NULL DEFAULT 1      COMMENT '状态:1=已发布,0=草稿',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '最后修改人ID(ORM自动填充)',
  `created_at`   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at`   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at`   DATETIME     DEFAULT NULL            COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  KEY `idx_type_receiver` (`type`,`receiver_id`),
  KEY `idx_sender` (`sender_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='消息通知';

-- ------------------------------------------------------------
-- 消息通知: 系统通知定向目标 (target_scope=2/3 时使用)
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `biz_message_target`;
CREATE TABLE `biz_message_target` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `message_id`  BIGINT UNSIGNED NOT NULL              COMMENT '消息ID',
  `target_type` TINYINT      NOT NULL                 COMMENT '目标类型:2=role,3=user',
  `target_id`   BIGINT UNSIGNED NOT NULL              COMMENT '角色ID或用户ID',
  PRIMARY KEY (`id`),
  KEY `idx_msg` (`message_id`),
  KEY `idx_tgt` (`target_type`,`target_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='消息通知定向目标';

-- ------------------------------------------------------------
-- 消息通知: 已读关系 (按用户独立记录)
--   兼具 "个人删除" 场景: hidden=1 表示从该用户的收件箱视角隐藏
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `biz_message_read`;
CREATE TABLE `biz_message_read` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `message_id` BIGINT UNSIGNED NOT NULL               COMMENT '消息ID',
  `user_id`    BIGINT UNSIGNED NOT NULL               COMMENT '用户ID',
  `read_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '阅读时间',
  `hidden`     TINYINT      NOT NULL DEFAULT 0       COMMENT '是否在收件箱视角隐藏(个人删除):1=是',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_msg_user` (`message_id`,`user_id`),
  KEY `idx_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='消息已读关系';

-- ------------------------------------------------------------
-- (公告模块已移除, 由「消息通知」中的「系统通知(全员)」覆盖其使用场景)
-- ------------------------------------------------------------

-- ============================================================
-- 种子数据
-- 默认账号: admin / 123456
-- bcrypt hash for "123456" (cost=10)
-- ============================================================
INSERT INTO `sys_user` (`id`,`username`,`password`,`nickname`,`status`,`must_change_pwd`)
VALUES (1, 'admin', '$2a$10$tUEhjYhhbY4OgzKvJRZZWexJRuKuaWFoKFb3U0PRnVjXpSBgvyEDK', '超级管理员', 1, 1);

INSERT INTO `sys_role` (`id`,`name`,`code`,`sort`,`status`,`remark`,`data_scope`) VALUES
  (1, '超级管理员', 'admin',  1, 1, '内置最高权限角色', 1),
  (2, '普通用户',   'common', 2, 1, '示例普通角色', 5);

-- 菜单(目录 + 仪表盘 + 个人中心 + 系统管理 + 公告)
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (30, 0, '仪表盘',   2, '/dashboard',      'dashboard',               'Odometer', 'dashboard:view',   1, 1, 1),
  -- 个人中心: visible=0 不在侧边栏展示 (仅从顶部头像下拉进入), 但仍在菜单管理表中可见可维护
  (40, 0, '个人中心', 2, '/profile',        'profile',                 'User',     'profile:view',     2, 0, 1),
  (1,  0, '系统管理', 1, '/system',         'Layout',                  'Setting',  'system',          10, 1, 1),
  (2,  0, '系统监控', 1, '/monitor',        'Layout',                  'Monitor',  'monitor',         20, 1, 1),
  (3,  0, '系统工具', 1, '/tool',           'Layout',                  'Tools',    'tool',            25, 1, 1),
  (10, 1, '用户管理', 2, '/system/users',   'system/users/index',      'User',     'system:user:list', 11, 1, 1),
  (11, 1, '角色管理', 2, '/system/roles',   'system/roles/index',      'UserFilled','system:role:list', 12, 1, 1),
  (12, 1, '菜单管理', 2, '/system/menus',   'system/menus/index',      'Menu',     'system:menu:list', 13, 1, 1),
  (13, 1, 'API管理',  2, '/system/apis',    'system/apis/index',       'Connection','system:api:list', 14, 1, 1),
  -- 按钮权限(visible=0, type=3)
  (101, 10, '用户新增', 3, '', '', '', 'system:user:create', 1, 0, 1),
  (102, 10, '用户修改', 3, '', '', '', 'system:user:update', 2, 0, 1),
  (103, 10, '用户删除', 3, '', '', '', 'system:user:delete', 3, 0, 1),
  (111, 11, '角色新增', 3, '', '', '', 'system:role:create', 1, 0, 1),
  (112, 11, '角色修改', 3, '', '', '', 'system:role:update', 2, 0, 1),
  (113, 11, '角色删除', 3, '', '', '', 'system:role:delete', 3, 0, 1),
  (114, 11, '角色授权', 3, '', '', '', 'system:role:assign', 4, 0, 1),
  (121, 12, '菜单新增', 3, '', '', '', 'system:menu:create', 1, 0, 1),
  (122, 12, '菜单修改', 3, '', '', '', 'system:menu:update', 2, 0, 1),
  (123, 12, '菜单删除', 3, '', '', '', 'system:menu:delete', 3, 0, 1),
  (131, 13, 'API新增', 3, '', '', '', 'system:api:create', 1, 0, 1),
  (132, 13, 'API修改', 3, '', '', '', 'system:api:update', 2, 0, 1),
  (133, 13, 'API删除', 3, '', '', '', 'system:api:delete', 3, 0, 1),
  -- 消息中心 (目录 + 子菜单 + 按钮权限)
  (50, 0,  '消息中心',  1, '/message',         'Layout',                  'ChatDotRound', 'message',                30, 1, 1),
  (51, 50, '系统通知',  2, '/message/system',  'message/system/index',    'Bell',         'message:system:list',    31, 1, 1),
  (52, 50, '私信通知',  2, '/message/private', 'message/private/index',   'Message',      'message:private:list',   32, 1, 1),
  (511, 51, '系统通知发布', 3, '', '', '', 'message:system:send',   1, 0, 1),
  (512, 51, '系统通知删除', 3, '', '', '', 'message:system:delete', 2, 0, 1),
  (521, 52, '私信发送',     3, '', '', '', 'message:private:send',  1, 0, 1),
  (522, 52, '私信删除',     3, '', '', '', 'message:delete',        2, 0, 1);

-- API接口资源初始数据
INSERT INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  -- 认证分组
  ('/api/v1/auth/login',     'POST', '认证', '用户登录'),
  ('/api/v1/auth/refresh',   'POST', '认证', '刷新Token'),
  ('/api/v1/auth/logout',    'POST', '认证', '用户登出'),
  ('/api/v1/auth/userInfo',  'GET',  '认证', '获取当前用户信息'),
  ('/api/v1/auth/menus',     'GET',  '认证', '获取当前用户菜单'),
  ('/api/v1/auth/profile',   'GET',  '认证', '个人中心详情'),
  ('/api/v1/auth/profile',   'PUT',  '认证', '修改个人资料'),
  ('/api/v1/auth/password',  'PUT',  '认证', '修改密码'),
  -- 用户管理
  ('/api/v1/system/users',         'GET',    '用户管理', '获取用户列表'),
  ('/api/v1/system/users',         'POST',   '用户管理', '创建用户'),
  ('/api/v1/system/users/:id',     'GET',    '用户管理', '获取用户详情'),
  ('/api/v1/system/users/:id',     'PUT',    '用户管理', '更新用户'),
  ('/api/v1/system/users/:id',     'DELETE', '用户管理', '删除用户'),
  ('/api/v1/system/users/:id/roles','PUT',   '用户管理', '设置用户角色'),
  -- 角色管理
  ('/api/v1/system/roles',         'GET',    '角色管理', '获取角色列表'),
  ('/api/v1/system/roles',         'POST',   '角色管理', '创建角色'),
  ('/api/v1/system/roles/:id',     'GET',    '角色管理', '获取角色详情'),
  ('/api/v1/system/roles/:id',     'PUT',    '角色管理', '更新角色'),
  ('/api/v1/system/roles/:id',     'DELETE', '角色管理', '删除角色'),
  ('/api/v1/system/roles/:id/menus','PUT',   '角色管理', '设置角色菜单'),
  ('/api/v1/system/roles/:id/menus','GET',   '角色管理', '获取角色菜单'),
  ('/api/v1/system/roles/:id/apis','PUT',    '角色管理', '设置角色API权限'),
  ('/api/v1/system/roles/:id/apis','GET',    '角色管理', '获取角色API权限'),
  -- 菜单管理
  ('/api/v1/system/menus',         'GET',    '菜单管理', '获取菜单列表'),
  ('/api/v1/system/menus',         'POST',   '菜单管理', '创建菜单'),
  ('/api/v1/system/menus/:id',     'GET',    '菜单管理', '获取菜单详情'),
  ('/api/v1/system/menus/:id',     'PUT',    '菜单管理', '更新菜单'),
  ('/api/v1/system/menus/:id',     'DELETE', '菜单管理', '删除菜单'),
  ('/api/v1/system/menus/tree',    'GET',    '菜单管理', '获取菜单树'),
  -- API管理
  ('/api/v1/system/apis',          'GET',    'API管理', '获取API列表'),
  ('/api/v1/system/apis',          'POST',   'API管理', '创建API'),
  ('/api/v1/system/apis/:id',      'GET',    'API管理', '获取API详情'),
  ('/api/v1/system/apis/:id',      'PUT',    'API管理', '更新API'),
  ('/api/v1/system/apis/:id',      'DELETE', 'API管理', '删除API'),
  -- 消息通知
  ('/api/v1/message',                'GET',    '消息通知', '管理-消息列表'),
  ('/api/v1/message/system',         'POST',   '消息通知', '发布系统通知'),
  ('/api/v1/message/private',        'POST',   '消息通知', '发送私信'),
  ('/api/v1/message/inbox',          'GET',    '消息通知', '我的收件箱'),
  ('/api/v1/message/inbox/:id',      'GET',    '消息通知', '阅读详情(自动已读)'),
  ('/api/v1/message/inbox/:id',      'DELETE', '消息通知', '从我的收件箱中删除'),
  ('/api/v1/message/unread-count',   'GET',    '消息通知', '未读数量'),
  ('/api/v1/message/read-all',       'PUT',    '消息通知', '全部标记为已读'),
  ('/api/v1/message/:id',            'GET',    '消息通知', '消息详情'),
  ('/api/v1/message/:id',            'DELETE', '消息通知', '管理员删除消息'),
  ('/api/v1/message/:id/read',       'PUT',    '消息通知', '标记为已读');

-- Casbin: 角色继承(g) + 策略(p)
-- g: 用户-角色映射, p: 角色-资源-操作 策略
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  -- g 策略: 用户 admin 属于 admin 角色
  ('g', 'admin', 'admin', '', '', '', ''),
  -- p 策略: admin 角色菜单权限(所有菜单含按钮)
  ('p', 'admin', 'menu:30',  'access', '', '', ''),
  ('p', 'admin', 'menu:40',  'access', '', '', ''),
  ('p', 'admin', 'menu:1',   'access', '', '', ''),
  ('p', 'admin', 'menu:2',   'access', '', '', ''),
  ('p', 'admin', 'menu:3',   'access', '', '', ''),
  ('p', 'admin', 'menu:10',  'access', '', '', ''),
  ('p', 'admin', 'menu:11',  'access', '', '', ''),
  ('p', 'admin', 'menu:12',  'access', '', '', ''),
  ('p', 'admin', 'menu:13',  'access', '', '', ''),
  ('p', 'admin', 'menu:101', 'access', '', '', ''),
  ('p', 'admin', 'menu:102', 'access', '', '', ''),
  ('p', 'admin', 'menu:103', 'access', '', '', ''),
  ('p', 'admin', 'menu:111', 'access', '', '', ''),
  ('p', 'admin', 'menu:112', 'access', '', '', ''),
  ('p', 'admin', 'menu:113', 'access', '', '', ''),
  ('p', 'admin', 'menu:114', 'access', '', '', ''),
  ('p', 'admin', 'menu:121', 'access', '', '', ''),
  ('p', 'admin', 'menu:122', 'access', '', '', ''),
  ('p', 'admin', 'menu:123', 'access', '', '', ''),
  ('p', 'admin', 'menu:131', 'access', '', '', ''),
  ('p', 'admin', 'menu:132', 'access', '', '', ''),
  ('p', 'admin', 'menu:133', 'access', '', '', ''),
  -- p 策略: admin 角色API权限(放行所有)
  ('p', 'admin', '/api/v1/*', '*', '', '', ''),
  -- p 策略: admin 消息中心菜单权限
  ('p', 'admin', 'menu:50',  'access', '', '', ''),
  ('p', 'admin', 'menu:51',  'access', '', '', ''),
  ('p', 'admin', 'menu:52',  'access', '', '', ''),
  ('p', 'admin', 'menu:511', 'access', '', '', ''),
  ('p', 'admin', 'menu:512', 'access', '', '', ''),
  ('p', 'admin', 'menu:521', 'access', '', '', ''),
  ('p', 'admin', 'menu:522', 'access', '', '', ''),
  -- p 策略: common 角色菜单权限(仪表盘 + 个人中心 + 消息中心)
  ('p', 'common', 'menu:30', 'access', '', '', ''),
  ('p', 'common', 'menu:40', 'access', '', '', ''),
  ('p', 'common', 'menu:50', 'access', '', '', ''),
  ('p', 'common', 'menu:52', 'access', '', '', ''),
  ('p', 'common', 'menu:521', 'access', '', '', ''),
  ('p', 'common', 'menu:522', 'access', '', '', ''),
  -- p 策略: common 角色API权限
  ('p', 'common', '/api/v1/auth/*', '*', '', '', ''),
  -- p 策略: common 消息通知 (收件箱 + 标记已读 + 发送私信 + 个人删除)
  ('p', 'common', '/api/v1/message/inbox',         'GET',    '', '', ''),
  ('p', 'common', '/api/v1/message/inbox/*',       'GET',    '', '', ''),
  ('p', 'common', '/api/v1/message/inbox/*',       'DELETE', '', '', ''),
  ('p', 'common', '/api/v1/message/unread-count',  'GET',    '', '', ''),
  ('p', 'common', '/api/v1/message/read-all',      'PUT',    '', '', ''),
  ('p', 'common', '/api/v1/message/*/read',        'PUT',    '', '', ''),
  ('p', 'common', '/api/v1/system/users',             'GET',    '', '', ''),
  ('p', 'common', '/api/v1/message/private',       'POST',   '', '', '');

-- ------------------------------------------------------------
-- 操作日志: 由中间件自动写入, 记录 POST/PUT/DELETE 操作(含成功/失败/未授权)
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `sys_audit_log`;
CREATE TABLE `sys_audit_log` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `user_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '用户ID',
  `username`    VARCHAR(64)  NOT NULL DEFAULT ''        COMMENT '用户名',
  `action`      VARCHAR(64)  NOT NULL DEFAULT ''        COMMENT '操作类型(create/update/delete/upload/login/...)',
  `resource`    VARCHAR(64)  NOT NULL DEFAULT ''        COMMENT '操作资源(如user/role/menu/dict/file)',
  `resource_id` VARCHAR(64)  NOT NULL DEFAULT ''        COMMENT '资源标识',
  `method`      VARCHAR(10)  NOT NULL DEFAULT ''        COMMENT 'HTTP方法',
  `path`        VARCHAR(255) NOT NULL DEFAULT ''        COMMENT '请求路径',
  `status_code` INT          NOT NULL DEFAULT 0        COMMENT 'HTTP状态码',
  `code`        INT          NOT NULL DEFAULT 0        COMMENT '业务码(0=成功)',
  `message`     VARCHAR(512) NOT NULL DEFAULT ''        COMMENT '业务消息/失败原因',
  `duration_ms` INT          NOT NULL DEFAULT 0        COMMENT '耗时(毫秒)',
  `request_id`  VARCHAR(64)  NOT NULL DEFAULT ''        COMMENT '请求ID(链路追踪)',
  `detail`      TEXT         NOT NULL                   COMMENT '详情(脱敏后的请求体)',
  `ip`          VARCHAR(64)  NOT NULL DEFAULT ''        COMMENT 'IP地址',
  `user_agent`  VARCHAR(512) NOT NULL DEFAULT ''        COMMENT 'User-Agent',
  `created_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_user` (`user_id`),
  KEY `idx_action` (`action`),
  KEY `idx_resource` (`resource`),
  KEY `idx_created` (`created_at`),
  KEY `idx_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='操作日志';

-- ------------------------------------------------------------
-- 字典类型主表
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `sys_dict_type`;
CREATE TABLE `sys_dict_type` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `type_code`  VARCHAR(64)  NOT NULL                   COMMENT '字典类型编码(唯一)',
  `type_name`  VARCHAR(128) NOT NULL                   COMMENT '字典类型名称',
  `status`     TINYINT      NOT NULL DEFAULT 1         COMMENT '状态:1=启用,0=禁用',
  `remark`     VARCHAR(255) NOT NULL DEFAULT ''         COMMENT '备注',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '最后修改人ID(ORM自动填充)',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME     DEFAULT NULL               COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_type_code` (`type_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='字典类型';

-- ------------------------------------------------------------
-- 字典数据子表
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `sys_dict_data`;
CREATE TABLE `sys_dict_data` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `type_id`    BIGINT UNSIGNED NOT NULL                COMMENT '字典类型ID(关联sys_dict_type)',
  `dict_label` VARCHAR(128) NOT NULL                   COMMENT '字典标签(展示名)',
  `dict_value` VARCHAR(255) NOT NULL DEFAULT ''         COMMENT '字典键值',
  `sort`       INT          NOT NULL DEFAULT 0         COMMENT '排序',
  `status`     TINYINT      NOT NULL DEFAULT 1         COMMENT '状态:1=启用,0=禁用',
  `remark`     VARCHAR(255) NOT NULL DEFAULT ''         COMMENT '备注',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '最后修改人ID(ORM自动填充)',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME     DEFAULT NULL               COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  KEY `idx_type` (`type_id`),
  KEY `idx_sort` (`sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='字典数据';

-- ------------------------------------------------------------
-- 文件管理
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `sys_file`;
CREATE TABLE `sys_file` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `name`          VARCHAR(255) NOT NULL DEFAULT ''        COMMENT '存储文件名',
  `original_name` VARCHAR(255) NOT NULL DEFAULT ''        COMMENT '原始文件名',
  `path`          VARCHAR(512) NOT NULL DEFAULT ''        COMMENT '存储路径',
  `url`           VARCHAR(512) NOT NULL DEFAULT ''        COMMENT '访问URL',
  `size`          BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '文件大小(字节)',
  `mime_type`     VARCHAR(128) NOT NULL DEFAULT ''        COMMENT 'MIME类型',
  `extension`     VARCHAR(32)  NOT NULL DEFAULT ''        COMMENT '文件扩展名',
  `user_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '上传用户ID',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '最后修改人ID(ORM自动填充)',
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `deleted_at`    DATETIME     DEFAULT NULL               COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  KEY `idx_user` (`user_id`),
  KEY `idx_ext` (`extension`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文件管理';

-- ------------------------------------------------------------
-- 新模块种子数据: 字典管理 / 文件管理 / 操作日志
-- ------------------------------------------------------------
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (14, 1, '字典管理', 2, '/system/dicts', 'system/dicts/index', 'List', 'system:dict:list', 15, 1, 1),
  (15, 1, '文件管理', 2, '/system/files', 'system/files/index', 'FolderOpened', 'system:file:list', 16, 1, 1),
  -- 操作日志挂在系统监控目录下 (id=2)
  (16, 2, '操作日志', 2, '/system/audit-logs', 'system/audit-logs/index', 'Timer', 'system:audit-log:list', 4, 1, 1);

-- 按钮权限
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (141, 14, '字典新增', 3, '', '', '', 'system:dict:create', 1, 0, 1),
  (142, 14, '字典修改', 3, '', '', '', 'system:dict:update', 2, 0, 1),
  (143, 14, '字典删除', 3, '', '', '', 'system:dict:delete', 3, 0, 1),
  (151, 15, '文件上传', 3, '', '', '', 'system:file:upload', 1, 0, 1),
  (152, 15, '文件删除', 3, '', '', '', 'system:file:delete', 2, 0, 1);

-- 新 API 资源
INSERT INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/audit-logs',    'GET',    '操作日志', '操作日志列表'),
  -- 字典类型管理
  ('/api/v1/system/dict-types',         'GET',    '字典管理', '字典类型列表'),
  ('/api/v1/system/dict-types',         'POST',   '字典管理', '新增字典类型'),
  ('/api/v1/system/dict-types/all',     'GET',    '字典管理', '字典类型全量'),
  ('/api/v1/system/dict-types/:id',     'PUT',    '字典管理', '修改字典类型'),
  ('/api/v1/system/dict-types/:id',     'DELETE', '字典管理', '删除字典类型'),
  -- 字典数据项管理
  ('/api/v1/system/dict-types/:typeId/items',       'GET',    '字典管理', '字典数据项列表'),
  ('/api/v1/system/dict-types/:typeId/items',       'POST',   '字典管理', '新增字典数据项'),
  ('/api/v1/system/dict-types/:typeId/items/:id',   'PUT',    '字典管理', '修改字典数据项'),
  ('/api/v1/system/dict-types/:typeId/items/:id',   'DELETE', '字典管理', '删除字典数据项'),
  ('/api/v1/system/dict-types/:typeId/items/sort',  'PUT',    '字典管理', '批量排序字典数据项'),
  -- 兼容: 全量字典数据
  ('/api/v1/system/dicts/all',     'GET',    '字典管理', '字典全量(按类型分组)'),
  -- 文件管理
  ('/api/v1/system/files',         'GET',    '文件管理', '文件列表'),
  ('/api/v1/system/files/upload',  'POST',   '文件管理', '上传文件'),
  ('/api/v1/system/files/:id',     'DELETE', '文件管理', '删除文件');

-- Casbin: admin 角色新菜单权限
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', 'admin', 'menu:14', 'access', '', '', ''),
  ('p', 'admin', 'menu:15', 'access', '', '', ''),
  ('p', 'admin', 'menu:16', 'access', '', '', ''),
  ('p', 'admin', 'menu:141', 'access', '', '', ''),
  ('p', 'admin', 'menu:142', 'access', '', '', ''),
  ('p', 'admin', 'menu:143', 'access', '', '', ''),
  ('p', 'admin', 'menu:151', 'access', '', '', ''),
  ('p', 'admin', 'menu:152', 'access', '', '', '');

-- ============================================================
-- 全局配置
-- ============================================================
DROP TABLE IF EXISTS `sys_config`;
CREATE TABLE `sys_config` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `config_key`  VARCHAR(128) NOT NULL                   COMMENT '配置键(唯一)',
  `config_value` TEXT         NOT NULL                  COMMENT '配置值',
  `config_type` TINYINT      NOT NULL DEFAULT 0         COMMENT '配置类型:0=文本,1=数字,2=布尔,3=JSON',
  `name`        VARCHAR(128) NOT NULL DEFAULT ''         COMMENT '配置名称(中文说明)',
  `remark`      VARCHAR(255) NOT NULL DEFAULT ''         COMMENT '备注',
  `status`      TINYINT      NOT NULL DEFAULT 1         COMMENT '状态:1=启用,0=禁用',
  `sort`        INT          NOT NULL DEFAULT 0         COMMENT '排序',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '最后修改人ID(ORM自动填充)',
  `created_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at`  DATETIME     DEFAULT NULL               COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_config_key` (`config_key`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='全局配置';

-- 初始种子数据
INSERT INTO `sys_config` (`config_key`, `config_value`, `config_type`, `name`, `remark`, `sort`) VALUES
  ('sys.name',        'Hinay Admin',        0, '系统名称',   '显示在登录页和浏览器标题', 1),
  ('sys.logo',        '',                   0, '系统Logo',   'Logo图片URL',              2),
  ('sys.copyright',   '© 2026 Hinay',       0, '版权信息',   '页脚版权文字',              3),
  ('sys.allow_register', 'false',           2, '开放注册',   '是否允许新用户自行注册',    4);

-- 菜单: 全局配置 (挂在系统管理目录下, id=1)
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (17, 1, '全局配置', 2, '/system/configs', 'system/configs/index', 'Tools', 'system:config:list', 17, 1, 1);

-- 按钮权限
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (171, 17, '配置新增', 3, '', '', '', 'system:config:create', 1, 0, 1),
  (172, 17, '配置修改', 3, '', '', '', 'system:config:update', 2, 0, 1),
  (173, 17, '配置删除', 3, '', '', '', 'system:config:delete', 3, 0, 1);

-- API 资源
INSERT INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/configs',      'GET',    '全局配置', '配置列表'),
  ('/api/v1/system/configs',      'POST',   '全局配置', '新增配置'),
  ('/api/v1/system/configs/all',  'GET',    '全局配置', '全部启用配置(前端使用)'),
  ('/api/v1/system/configs/:id',  'PUT',    '全局配置', '修改配置'),
  ('/api/v1/system/configs/:id',  'DELETE', '全局配置', '删除配置');

-- Casbin 权限
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', 'admin', 'menu:17',  'access', '', '', ''),
  ('p', 'admin', 'menu:171', 'access', '', '', ''),
  ('p', 'admin', 'menu:172', 'access', '', '', ''),
  ('p', 'admin', 'menu:173', 'access', '', '', '');

-- ============================================================
-- 组织机构管理
-- ============================================================

-- 菜单: 组织机构 (挂在系统管理目录下, id=1)
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (60, 1, '组织机构', 2, '/system/orgs', 'system/orgs/index', 'OfficeBuilding', 'system:org:list', 10, 1, 1);

-- 按钮权限
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (601, 60, '组织新增', 3, '', '', '', 'system:org:create', 1, 0, 1),
  (602, 60, '组织修改', 3, '', '', '', 'system:org:update', 2, 0, 1),
  (603, 60, '组织删除', 3, '', '', '', 'system:org:delete', 3, 0, 1);

-- API 资源
INSERT INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/orgs',      'GET',    '组织机构', '组织列表/树'),
  ('/api/v1/system/orgs',      'POST',   '组织机构', '创建组织'),
  ('/api/v1/system/orgs/:id',  'GET',    '组织机构', '组织详情'),
  ('/api/v1/system/orgs/:id',  'PUT',    '组织机构', '更新组织'),
  ('/api/v1/system/orgs/:id',  'DELETE', '组织机构', '删除组织'),
  ('/api/v1/system/orgs/tree', 'GET',    '组织机构', '组织树');

-- Casbin 权限
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', 'admin', 'menu:60',  'access', '', '', ''),
  ('p', 'admin', 'menu:601', 'access', '', '', ''),
  ('p', 'admin', 'menu:602', 'access', '', '', ''),
  ('p', 'admin', 'menu:603', 'access', '', '', '');

-- ============================================================
-- 登录日志
-- ============================================================
DROP TABLE IF EXISTS `sys_login_log`;
CREATE TABLE `sys_login_log` (
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

-- 菜单: 登录日志 (挂在系统监控目录下, id=2)
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (18, 2, '登录日志', 2, '/system/login-logs', 'system/login-logs/index', 'Key', 'system:login-log:list', 3, 1, 1);

-- 按钮权限
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (181, 18, '登录日志删除', 3, '', '', '', 'system:login-log:delete', 1, 0, 1);

-- API 资源
INSERT INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/login-logs',      'GET',    '登录日志', '登录日志列表'),
  ('/api/v1/system/login-logs/:id',  'DELETE', '登录日志', '删除登录日志');

-- Casbin: admin 角色
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', 'admin', 'menu:18',  'access', '', '', ''),
  ('p', 'admin', 'menu:181', 'access', '', '', '');

-- ============================================================
-- 在线用户
-- ============================================================
-- 会话数据存 Redis (hinay:online:sessions), 无需建表。

-- 菜单: 在线用户 (挂在系统监控目录下, id=2; 图标改 View 避免与目录图标重复)
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (19, 2, '在线用户', 2, '/system/online', 'system/online/index', 'View', 'system:online:list', 1, 1, 1);

-- 按钮权限
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (191, 19, '强制下线', 3, '', '', '', 'system:online:kick', 1, 0, 1);

-- API 资源
INSERT INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/online',     'GET',    '在线用户', '在线用户列表'),
  ('/api/v1/system/online/:id', 'DELETE', '在线用户', '强制下线');

-- Casbin: admin 角色
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', 'admin', 'menu:19',  'access', '', '', ''),
  ('p', 'admin', 'menu:191', 'access', '', '', '');

-- ============================================================
-- 定时任务
-- ============================================================
DROP TABLE IF EXISTS `sys_job`;
CREATE TABLE `sys_job` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '任务ID',
  `name`       VARCHAR(64)  NOT NULL                COMMENT '任务名称',
  `handler`    VARCHAR(128) NOT NULL                COMMENT '处理器名称(需已注册)',
  `cron_expr`  VARCHAR(64)  NOT NULL                COMMENT 'cron表达式(6位: 秒 分 时 日 月 周)',
  `params`     VARCHAR(512) NOT NULL DEFAULT ''     COMMENT '任务参数(JSON, 可空)',
  `status`     TINYINT      NOT NULL DEFAULT 0      COMMENT '状态:1=启动,0=暂停',
  `remark`     VARCHAR(255) NOT NULL DEFAULT ''     COMMENT '备注',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '最后修改人ID(ORM自动填充)',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME     DEFAULT NULL            COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='定时任务';

DROP TABLE IF EXISTS `sys_job_log`;
CREATE TABLE `sys_job_log` (
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

-- 菜单: 定时任务 (挂在系统监控目录下, id=2)
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (20, 2, '定时任务', 2, '/system/jobs', 'system/jobs/index', 'AlarmClock', 'system:job:list', 2, 1, 1);

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

-- ============================================================
-- Excel 导入导出 (用户模块示例)
-- ============================================================
INSERT INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/users/export',           'GET',  '用户管理', '用户列表导出'),
  ('/api/v1/system/users/import',           'POST', '用户管理', '用户导入'),
  ('/api/v1/system/users/import-template',  'GET',  '用户管理', '用户导入模板下载');

-- ============================================================
-- 密码策略 (配置驱动: 全局配置页可直接修改, 无需重启)
-- ============================================================
INSERT INTO `sys_config` (`config_key`, `config_value`, `config_type`, `name`, `remark`, `sort`) VALUES
  ('sys.password.min_length',      '6',  1, '密码最小长度', '设置新密码时的最小长度', 10),
  ('sys.password.max_length',      '32', 1, '密码最大长度', '设置新密码时的最大长度', 11),
  ('sys.password.require_upper',   'false', 2, '密码需含大写字母', '设置新密码时校验', 12),
  ('sys.password.require_lower',   'false', 2, '密码需含小写字母', '设置新密码时校验', 13),
  ('sys.password.require_digit',   'false', 2, '密码需含数字',     '设置新密码时校验', 14),
  ('sys.password.require_special', 'false', 2, '密码需含特殊字符', '设置新密码时校验', 15),
  ('sys.password.expire_days',     '0',  1, '密码有效期(天)', '0=永不过期; 过期后登录强制改密', 16);

-- ============================================================
-- 网页版代码生成 (gencode)
-- ============================================================
-- 菜单: 代码生成 (挂在系统工具目录下, id=3)
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (21, 3, '代码生成', 2, '/system/gencode', 'system/gencode/index', 'MagicStick', 'system:gencode:list', 1, 1, 1);

-- 按钮权限
INSERT INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (211, 21, '生成预览', 3, '', '', '', 'system:gencode:preview', 1, 0, 1),
  (212, 21, '打包下载', 3, '', '', '', 'system:gencode:download', 2, 0, 1),
  (213, 21, '写入源码', 3, '', '', '', 'system:gencode:write', 3, 0, 1);

-- API 资源
INSERT INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/gencode/tables',    'GET',  '代码生成', '可生成表清单'),
  ('/api/v1/system/gencode/columns',   'GET',  '代码生成', '表列信息'),
  ('/api/v1/system/gencode/preview',   'POST', '代码生成', '预览生成代码'),
  ('/api/v1/system/gencode/download',  'GET',  '代码生成', '下载生成代码(zip)'),
  ('/api/v1/system/gencode/write',     'POST', '代码生成', '生成并写入源码树');

-- Casbin: admin 角色
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', 'admin', 'menu:21',  'access', '', '', ''),
  ('p', 'admin', 'menu:211', 'access', '', '', ''),
  ('p', 'admin', 'menu:212', 'access', '', '', ''),
  ('p', 'admin', 'menu:213', 'access', '', '', '');

-- ============================================================
-- TOTP 两步验证 (2FA)
-- ============================================================
DROP TABLE IF EXISTS `sys_user_totp`;
CREATE TABLE `sys_user_totp` (
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
-- 说明: 本表无 create_id/update_id 审计列(行为人即 user_id 本身), 不纳入 ormfill 白名单;
-- 解绑走物理删除(Unscoped), 避免 uk_user_id 与软删残留行冲突。

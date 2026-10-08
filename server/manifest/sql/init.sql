-- ============================================================
-- Hinay Admin 数据库初始化脚本 (新装唯一入口)
-- 数据库: hinay_admin (MySQL 8+, utf8mb4)
--
-- 本文件为全量基线: 基础库 + 全部模块 (审批流/岗位/AI/微信/请假Demo)
-- 与其种子一次到位, 新装只需执行本文件 (make initdb)。
-- 原 upgrade/ (社区版存量增量) 与 upgrade-modules/ (模块增量 p001~p018)
-- 的最终态均已并入, 目录已移除 —— 项目默认面向新装应用, 不再维护
-- 存量库迁移脚本。
-- 可选演示种子在 demo/ 目录 (流程测试账号/演示定义, 生产库不执行)。
-- 幂等: 建表 IF NOT EXISTS / 种子 INSERT IGNORE, 可重复执行。
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
CREATE TABLE IF NOT EXISTS `sys_user` (
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
CREATE TABLE IF NOT EXISTS `sys_org` (
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
CREATE TABLE IF NOT EXISTS `sys_role` (
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
CREATE TABLE IF NOT EXISTS `sys_role_org` (
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
CREATE TABLE IF NOT EXISTS `sys_menu` (
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
CREATE TABLE IF NOT EXISTS `sys_api` (
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
-- 字段语义 (关联键均为ID, 与用户名/角色code解耦, 改名不影响权限):
--   ptype = p  (策略)        : v0=sub(角色ID), v1=obj(path 或 menu:<菜单ID>), v2=act(method), v3~v5 预留
--   ptype = g  (角色继承/分组): v0=用户ID, v1=角色ID,        v2=domain (可选)
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `casbin_rule`;
CREATE TABLE IF NOT EXISTS `casbin_rule` (
  `id`    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT             COMMENT '主键',
  `ptype` VARCHAR(100) NOT NULL DEFAULT ''                    COMMENT '策略类型: p / g / g2 ...',
  `v0`    VARCHAR(100) NOT NULL DEFAULT ''                    COMMENT 'p:sub(角色ID) | g:用户ID',
  `v1`    VARCHAR(100) NOT NULL DEFAULT ''                    COMMENT 'p:obj(path/menu:菜单ID) | g:角色ID',
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
CREATE TABLE IF NOT EXISTS `biz_message` (
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
CREATE TABLE IF NOT EXISTS `biz_message_target` (
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
CREATE TABLE IF NOT EXISTS `biz_message_read` (
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
INSERT IGNORE INTO `sys_user` (`id`,`username`,`password`,`nickname`,`status`,`must_change_pwd`)
VALUES (1, 'admin', '$2a$10$tUEhjYhhbY4OgzKvJRZZWexJRuKuaWFoKFb3U0PRnVjXpSBgvyEDK', '超级管理员', 1, 1);

INSERT IGNORE INTO `sys_role` (`id`,`name`,`code`,`sort`,`status`,`remark`,`data_scope`) VALUES
  (1, '超级管理员', 'admin',  1, 1, '内置最高权限角色', 1),
  (2, '普通用户',   'common', 2, 1, '示例普通角色', 5);

-- 菜单(目录 + 仪表盘 + 个人中心 + 系统管理 + 公告)
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
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
INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  -- 认证分组
  ('/api/v1/auth/login',     'POST', '认证', '用户登录'),
  ('/api/v1/auth/refresh',   'POST', '认证', '刷新Token'),
  ('/api/v1/auth/logout',    'POST', '认证', '用户登出'),
  ('/api/v1/auth/userInfo',  'GET',  '认证', '获取当前用户信息'),
  ('/api/v1/auth/menus',     'GET',  '认证', '获取当前用户菜单'),
  ('/api/v1/auth/profile',   'GET',  '认证', '个人中心详情'),
  ('/api/v1/auth/profile',   'PUT',  '认证', '修改个人资料'),
  ('/api/v1/auth/password',  'PUT',  '认证', '修改密码'),
  ('/api/v1/auth/captcha',         'GET',  '认证', '获取图形验证码(公开)'),
  ('/api/v1/auth/register',        'POST', '认证', '用户注册(公开, 受 sys.allow_register 开关控制)'),
  ('/api/v1/auth/register/status', 'GET',  '认证', '查询注册开关(公开)'),
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
  ('/api/v1/message/:id/read',       'PUT',    '消息通知', '标记为已读'),
  -- 历史缺漏补种 (tools/genapi -check 扫描发现: TOTP/SSE/头像/全量下拉/重置密码)
  ('/api/v1/auth/avatar',                'POST', '认证',     '上传头像'),
  ('/api/v1/auth/login/totp',            'POST', '认证',     '两步验证登录'),
  ('/api/v1/auth/public-key',            'GET',  '认证',     '获取登录加密公钥'),
  ('/api/v1/auth/totp/disable',          'PUT',  '认证',     '解绑两步验证'),
  ('/api/v1/auth/totp/enable',           'PUT',  '认证',     '绑定两步验证'),
  ('/api/v1/auth/totp/setup',            'GET',  '认证',     '生成两步验证密钥'),
  ('/api/v1/message/events',             'GET',  '消息通知', '消息事件流(SSE)'),
  ('/api/v1/system/apis/all',            'GET',  'API管理',  '全量API'),
  ('/api/v1/system/roles/all',           'GET',  '角色管理', '全量角色'),
  ('/api/v1/system/users/{id}/password', 'PUT',  '用户管理', '重置密码');

-- Casbin: 角色继承(g) + 策略(p)
-- g: 用户ID-角色ID 映射, p: 角色ID-资源-操作 策略 (关联键均为ID)
INSERT IGNORE INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  -- g 策略: 用户1(admin) 属于角色1(admin, 内置超管)
  ('g', '1', '1', '', '', '', ''),
  -- p 策略: admin 角色菜单权限(所有菜单含按钮)
  ('p', '1', 'menu:30',  'access', '', '', ''),
  ('p', '1', 'menu:40',  'access', '', '', ''),
  ('p', '1', 'menu:1',   'access', '', '', ''),
  ('p', '1', 'menu:2',   'access', '', '', ''),
  ('p', '1', 'menu:3',   'access', '', '', ''),
  ('p', '1', 'menu:10',  'access', '', '', ''),
  ('p', '1', 'menu:11',  'access', '', '', ''),
  ('p', '1', 'menu:12',  'access', '', '', ''),
  ('p', '1', 'menu:13',  'access', '', '', ''),
  ('p', '1', 'menu:101', 'access', '', '', ''),
  ('p', '1', 'menu:102', 'access', '', '', ''),
  ('p', '1', 'menu:103', 'access', '', '', ''),
  ('p', '1', 'menu:111', 'access', '', '', ''),
  ('p', '1', 'menu:112', 'access', '', '', ''),
  ('p', '1', 'menu:113', 'access', '', '', ''),
  ('p', '1', 'menu:114', 'access', '', '', ''),
  ('p', '1', 'menu:121', 'access', '', '', ''),
  ('p', '1', 'menu:122', 'access', '', '', ''),
  ('p', '1', 'menu:123', 'access', '', '', ''),
  ('p', '1', 'menu:131', 'access', '', '', ''),
  ('p', '1', 'menu:132', 'access', '', '', ''),
  ('p', '1', 'menu:133', 'access', '', '', ''),
  -- p 策略: admin 角色API权限(放行所有)
  ('p', '1', '/api/v1/*', '*', '', '', ''),
  -- p 策略: admin 消息中心菜单权限
  ('p', '1', 'menu:50',  'access', '', '', ''),
  ('p', '1', 'menu:51',  'access', '', '', ''),
  ('p', '1', 'menu:52',  'access', '', '', ''),
  ('p', '1', 'menu:511', 'access', '', '', ''),
  ('p', '1', 'menu:512', 'access', '', '', ''),
  ('p', '1', 'menu:521', 'access', '', '', ''),
  ('p', '1', 'menu:522', 'access', '', '', ''),
  -- p 策略: common 角色菜单权限(仪表盘 + 个人中心 + 消息中心)
  ('p', '2', 'menu:30', 'access', '', '', ''),
  ('p', '2', 'menu:40', 'access', '', '', ''),
  ('p', '2', 'menu:50', 'access', '', '', ''),
  ('p', '2', 'menu:52', 'access', '', '', ''),
  ('p', '2', 'menu:521', 'access', '', '', ''),
  ('p', '2', 'menu:522', 'access', '', '', ''),
  -- p 策略: common 角色API权限
  ('p', '2', '/api/v1/auth/*', '*', '', '', ''),
  -- p 策略: common 消息通知 (收件箱 + 标记已读 + 发送私信 + 个人删除)
  ('p', '2', '/api/v1/message/inbox',         'GET',    '', '', ''),
  ('p', '2', '/api/v1/message/inbox/*',       'GET',    '', '', ''),
  ('p', '2', '/api/v1/message/inbox/*',       'DELETE', '', '', ''),
  ('p', '2', '/api/v1/message/unread-count',  'GET',    '', '', ''),
  ('p', '2', '/api/v1/message/read-all',      'PUT',    '', '', ''),
  ('p', '2', '/api/v1/message/*/read',        'PUT',    '', '', ''),
  ('p', '2', '/api/v1/system/users',             'GET',    '', '', ''),
  ('p', '2', '/api/v1/message/private',       'POST',   '', '', '');

-- ------------------------------------------------------------
-- 操作日志: 由中间件自动写入, 记录 POST/PUT/DELETE 操作(含成功/失败/未授权)
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `sys_audit_log`;
CREATE TABLE IF NOT EXISTS `sys_audit_log` (
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
CREATE TABLE IF NOT EXISTS `sys_dict_type` (
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
CREATE TABLE IF NOT EXISTS `sys_dict_data` (
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
CREATE TABLE IF NOT EXISTS `sys_file` (
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
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (14, 1, '字典管理', 2, '/system/dicts', 'system/dicts/index', 'List', 'system:dict:list', 15, 1, 1),
  (15, 1, '文件管理', 2, '/system/files', 'system/files/index', 'FolderOpened', 'system:file:list', 16, 1, 1),
  -- 操作日志挂在系统监控目录下 (id=2)
  (16, 2, '操作日志', 2, '/system/audit-logs', 'system/audit-logs/index', 'Timer', 'system:audit-log:list', 4, 1, 1);

-- 按钮权限
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (141, 14, '字典新增', 3, '', '', '', 'system:dict:create', 1, 0, 1),
  (142, 14, '字典修改', 3, '', '', '', 'system:dict:update', 2, 0, 1),
  (143, 14, '字典删除', 3, '', '', '', 'system:dict:delete', 3, 0, 1),
  (151, 15, '文件上传', 3, '', '', '', 'system:file:upload', 1, 0, 1),
  (152, 15, '文件删除', 3, '', '', '', 'system:file:delete', 2, 0, 1);

-- 新 API 资源
INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
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
INSERT IGNORE INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', '1', 'menu:14', 'access', '', '', ''),
  ('p', '1', 'menu:15', 'access', '', '', ''),
  ('p', '1', 'menu:16', 'access', '', '', ''),
  ('p', '1', 'menu:141', 'access', '', '', ''),
  ('p', '1', 'menu:142', 'access', '', '', ''),
  ('p', '1', 'menu:143', 'access', '', '', ''),
  ('p', '1', 'menu:151', 'access', '', '', ''),
  ('p', '1', 'menu:152', 'access', '', '', '');

-- ============================================================
-- 全局配置
-- ============================================================
DROP TABLE IF EXISTS `sys_config`;
CREATE TABLE IF NOT EXISTS `sys_config` (
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
INSERT IGNORE INTO `sys_config` (`config_key`, `config_value`, `config_type`, `name`, `remark`, `sort`) VALUES
  ('sys.name',        'Hinay Admin',        0, '系统名称',   '显示在登录页和浏览器标题', 1),
  ('sys.logo',        '',                   0, '系统Logo',   'Logo图片URL',              2),
  ('sys.copyright',   '© 2026 Hinay',       0, '版权信息',   '页脚版权文字',              3),
  ('sys.allow_register', 'false',           2, '开放注册',   '是否允许新用户自行注册',    4),
  ('sys.captcha_enable', 'true',            2, '登录验证码', '登录/注册页图形验证码开关', 5);

-- 菜单: 全局配置 (挂在系统管理目录下, id=1)
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (17, 1, '全局配置', 2, '/system/configs', 'system/configs/index', 'Tools', 'system:config:list', 17, 1, 1);

-- 按钮权限
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (171, 17, '配置新增', 3, '', '', '', 'system:config:create', 1, 0, 1),
  (172, 17, '配置修改', 3, '', '', '', 'system:config:update', 2, 0, 1),
  (173, 17, '配置删除', 3, '', '', '', 'system:config:delete', 3, 0, 1);

-- API 资源
INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/configs',      'GET',    '全局配置', '配置列表'),
  ('/api/v1/system/configs',      'POST',   '全局配置', '新增配置'),
  ('/api/v1/system/configs/all',  'GET',    '全局配置', '全部启用配置(前端使用)'),
  ('/api/v1/system/configs/:id',  'PUT',    '全局配置', '修改配置'),
  ('/api/v1/system/configs/:id',  'DELETE', '全局配置', '删除配置');

-- Casbin 权限
INSERT IGNORE INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', '1', 'menu:17',  'access', '', '', ''),
  ('p', '1', 'menu:171', 'access', '', '', ''),
  ('p', '1', 'menu:172', 'access', '', '', ''),
  ('p', '1', 'menu:173', 'access', '', '', '');

-- ============================================================
-- 组织机构管理
-- ============================================================

-- 菜单: 组织机构 (挂在系统管理目录下, id=1)
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (60, 1, '组织机构', 2, '/system/orgs', 'system/orgs/index', 'OfficeBuilding', 'system:org:list', 10, 1, 1);

-- 按钮权限
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (601, 60, '组织新增', 3, '', '', '', 'system:org:create', 1, 0, 1),
  (602, 60, '组织修改', 3, '', '', '', 'system:org:update', 2, 0, 1),
  (603, 60, '组织删除', 3, '', '', '', 'system:org:delete', 3, 0, 1);

-- API 资源
INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/orgs',      'GET',    '组织机构', '组织列表/树'),
  ('/api/v1/system/orgs',      'POST',   '组织机构', '创建组织'),
  ('/api/v1/system/orgs/:id',  'GET',    '组织机构', '组织详情'),
  ('/api/v1/system/orgs/:id',  'PUT',    '组织机构', '更新组织'),
  ('/api/v1/system/orgs/:id',  'DELETE', '组织机构', '删除组织'),
  ('/api/v1/system/orgs/tree', 'GET',    '组织机构', '组织树');

-- Casbin 权限
INSERT IGNORE INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', '1', 'menu:60',  'access', '', '', ''),
  ('p', '1', 'menu:601', 'access', '', '', ''),
  ('p', '1', 'menu:602', 'access', '', '', ''),
  ('p', '1', 'menu:603', 'access', '', '', '');

-- ============================================================
-- 登录日志
-- ============================================================
DROP TABLE IF EXISTS `sys_login_log`;
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

-- 菜单: 登录日志 (挂在系统监控目录下, id=2)
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (18, 2, '登录日志', 2, '/system/login-logs', 'system/login-logs/index', 'Key', 'system:login-log:list', 3, 1, 1);

-- 按钮权限
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (181, 18, '登录日志删除', 3, '', '', '', 'system:login-log:delete', 1, 0, 1);

-- API 资源
INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/login-logs',      'GET',    '登录日志', '登录日志列表'),
  ('/api/v1/system/login-logs/:id',  'DELETE', '登录日志', '删除登录日志');

-- Casbin: admin 角色
INSERT IGNORE INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', '1', 'menu:18',  'access', '', '', ''),
  ('p', '1', 'menu:181', 'access', '', '', '');

-- ============================================================
-- 在线用户
-- ============================================================
-- 会话数据存 Redis (hinay:online:sessions), 无需建表。

-- 菜单: 在线用户 (挂在系统监控目录下, id=2; 图标改 View 避免与目录图标重复)
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (19, 2, '在线用户', 2, '/system/online', 'system/online/index', 'View', 'system:online:list', 1, 1, 1);

-- 按钮权限
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (191, 19, '强制下线', 3, '', '', '', 'system:online:kick', 1, 0, 1);

-- API 资源
INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/online',     'GET',    '在线用户', '在线用户列表'),
  ('/api/v1/system/online/:id', 'DELETE', '在线用户', '强制下线');

-- Casbin: admin 角色
INSERT IGNORE INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', '1', 'menu:19',  'access', '', '', ''),
  ('p', '1', 'menu:191', 'access', '', '', '');

-- ============================================================
-- 定时任务
-- ============================================================
DROP TABLE IF EXISTS `sys_job`;
CREATE TABLE IF NOT EXISTS `sys_job` (
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

-- 菜单: 定时任务 (挂在系统监控目录下, id=2)
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (20, 2, '定时任务', 2, '/system/jobs', 'system/jobs/index', 'AlarmClock', 'system:job:list', 2, 1, 1);

-- 按钮权限
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (201, 20, '任务新增', 3, '', '', '', 'system:job:create', 1, 0, 1),
  (202, 20, '任务修改', 3, '', '', '', 'system:job:update', 2, 0, 1),
  (203, 20, '任务删除', 3, '', '', '', 'system:job:delete', 3, 0, 1),
  (204, 20, '任务启停', 3, '', '', '', 'system:job:status', 4, 0, 1),
  (205, 20, '立即执行', 3, '', '', '', 'system:job:run', 5, 0, 1),
  (206, 20, '执行日志', 3, '', '', '', 'system:job:log', 6, 0, 1);

-- API 资源
INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/jobs',              'GET',    '定时任务', '任务列表'),
  ('/api/v1/system/jobs',              'POST',   '定时任务', '新增任务'),
  ('/api/v1/system/jobs/:id',          'PUT',    '定时任务', '修改任务'),
  ('/api/v1/system/jobs/:id',          'DELETE', '定时任务', '删除任务'),
  ('/api/v1/system/jobs/:id/status',   'PUT',    '定时任务', '启动/暂停任务'),
  ('/api/v1/system/jobs/:id/run',      'POST',   '定时任务', '立即执行一次'),
  ('/api/v1/system/jobs/logs',         'GET',    '定时任务', '执行日志列表'),
  ('/api/v1/system/jobs/handlers',     'GET',    '定时任务', '已注册处理器列表');

-- Casbin: admin 角色
INSERT IGNORE INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', '1', 'menu:20',  'access', '', '', ''),
  ('p', '1', 'menu:201', 'access', '', '', ''),
  ('p', '1', 'menu:202', 'access', '', '', ''),
  ('p', '1', 'menu:203', 'access', '', '', ''),
  ('p', '1', 'menu:204', 'access', '', '', ''),
  ('p', '1', 'menu:205', 'access', '', '', ''),
  ('p', '1', 'menu:206', 'access', '', '', '');

-- ============================================================
-- 服务器监控 (挂在系统监控目录下, 仅超管可见)
-- ============================================================

-- 菜单: 服务器监控 (路径走 /system 前缀, 复用前端 /system 路由守卫;
-- 权限双层: 种子只给 admin 角色授权 + 逻辑层 contextx.IsAdmin 硬校验)
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (21, 2, '服务器监控', 2, '/system/monitor', 'system/monitor/index', 'Cpu', 'monitor:server:list', 5, 1, 1);

-- API 资源 (纯查看页, 无按钮权限)
INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/monitor/server', 'GET', '服务器监控', '服务器指标(CPU/内存/磁盘/Go运行时, 仅超管)'),
  ('/api/v1/system/monitor/mysql',  'GET', '服务器监控', 'MySQL运行状态(仅超管)'),
  ('/api/v1/system/monitor/redis',  'GET', '服务器监控', 'Redis运行状态(仅超管)');

-- Casbin: 仅 admin 角色授予菜单 (API 侧 admin 已有 /api/v1/* 通配策略)
INSERT IGNORE INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', '1', 'menu:21', 'access', '', '', '');

-- ============================================================
-- Excel 导入导出 (用户模块示例)
-- ============================================================
INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/users/export',           'GET',  '用户管理', '用户列表导出'),
  ('/api/v1/system/users/import',           'POST', '用户管理', '用户导入'),
  ('/api/v1/system/users/import-template',  'GET',  '用户管理', '用户导入模板下载');

-- ============================================================
-- 密码策略 (配置驱动: 全局配置页可直接修改, 无需重启)
-- ============================================================
INSERT IGNORE INTO `sys_config` (`config_key`, `config_value`, `config_type`, `name`, `remark`, `sort`) VALUES
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
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (21, 3, '代码生成', 2, '/system/gencode', 'system/gencode/index', 'MagicStick', 'system:gencode:list', 1, 1, 1);

-- 按钮权限
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (211, 21, '生成预览', 3, '', '', '', 'system:gencode:preview', 1, 0, 1),
  (212, 21, '打包下载', 3, '', '', '', 'system:gencode:download', 2, 0, 1),
  (213, 21, '写入源码', 3, '', '', '', 'system:gencode:write', 3, 0, 1);

-- API 资源
INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/system/gencode/tables',    'GET',  '代码生成', '可生成表清单'),
  ('/api/v1/system/gencode/columns',   'GET',  '代码生成', '表列信息'),
  ('/api/v1/system/gencode/preview',   'POST', '代码生成', '预览生成代码'),
  ('/api/v1/system/gencode/download',  'GET',  '代码生成', '下载生成代码(zip)'),
  ('/api/v1/system/gencode/write',     'POST', '代码生成', '生成并写入源码树');

-- Casbin: admin 角色
INSERT IGNORE INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', '1', 'menu:21',  'access', '', '', ''),
  ('p', '1', 'menu:211', 'access', '', '', ''),
  ('p', '1', 'menu:212', 'access', '', '', ''),
  ('p', '1', 'menu:213', 'access', '', '', '');

-- ============================================================
-- TOTP 两步验证 (2FA)
-- ============================================================
DROP TABLE IF EXISTS `sys_user_totp`;
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
-- 说明: 本表无 create_id/update_id 审计列(行为人即 user_id 本身), 不纳入 ormfill 白名单;
-- 解绑走物理删除(Unscoped), 避免 uk_user_id 与软删残留行冲突。

-- ############################################################
-- 模块: 审批流 / AI 对话 / 微信公众号 / 业务审批 Demo
-- (原 upgrade-modules/modules_init.sql 全量并入, 面向新装; 历史增量
--  脚本 p001~p018 的最终态均已含, 存量库迁移脚本不再单独保留)
-- 可选演示种子: demo/p012_flow_demo_test_seed.sql (测试账号+演示流程),
--              demo/p016_biz_leave_demo.sql (请假Demo演示定义+授权)。
-- ############################################################
-- ############################################################
-- 一、自由审批流 + 岗位管理 (原 p001/p002)
-- 模型: 单行定义(发布原地生效) + 实例携带表单/节点树快照(在途不受定义后续修改影响)
--       + 任务按"节点×审批人"落行; 岗位管理为审批"指定岗位"与"部门主管"解析依据。
-- 说明: wf_task / wf_record 为行为表, 不建审计列, 不纳入 ormfill;
--       wf_definition / wf_instance / sys_post 已在 ormfill 登记; sys_user_post 纯关联表豁免。
-- 菜单使用 9000 号段。
-- ############################################################

-- ------------------------------------------------------------
-- 流程定义 (版本行: version=0 草稿, >=1 已发布版本; 同 flow_key 多版本)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `wf_definition` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `flow_key`   VARCHAR(64)  NOT NULL DEFAULT ''     COMMENT '流程标识(同标识共用一组版本, 如 leave)',
  `name`       VARCHAR(128) NOT NULL                COMMENT '流程名称',
  `form_conf`  JSON         NULL                   COMMENT '表单字段定义 JSON [{key,label,type,options,required}]',
  `flow_conf`  JSON         NULL                   COMMENT '节点树定义 JSON {id,type,name,child,...}',
  `version`    INT          NOT NULL DEFAULT 0      COMMENT '版本号=发布次数,0=未发布过',
  `status`     TINYINT      NOT NULL DEFAULT 0      COMMENT '状态:0=草稿,1=已发布,2=已停用',
  `remark`     VARCHAR(255) NOT NULL DEFAULT ''     COMMENT '备注',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '最后修改人ID(ORM自动填充)',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME     DEFAULT NULL            COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  KEY `idx_flow_key` (`flow_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='审批流程定义';

-- ------------------------------------------------------------
-- 流程实例 (引用具体定义版本行, 定义后续发布新版本不影响在途实例)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `wf_instance` (
  `id`                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `definition_id`     BIGINT UNSIGNED NOT NULL             COMMENT '定义版本行ID',
  `flow_key`          VARCHAR(64)  NOT NULL DEFAULT ''     COMMENT '流程标识(冗余)',
  `flow_name`         VARCHAR(128) NOT NULL DEFAULT ''     COMMENT '流程名称(冗余)',
  `biz_id`            BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '业务关联ID(0=审批中心直接发起)',
  `title`             VARCHAR(128) NOT NULL                COMMENT '申请标题',
  `form_data`         JSON         NULL                   COMMENT '提交的表单数据 JSON',
  `form_conf`         JSON         NULL                   COMMENT '表单定义快照(发起时从定义复制)',
  `flow_conf`         JSON         NULL                   COMMENT '节点树快照(发起时从定义复制, 驳回重提沿用)',
  `self_selects`      JSON         NULL                   COMMENT '发起人自选审批人快照 {nodeId: [userId]} (发起/重提时写入, 推进时读取)',
  `current_node_ids`  VARCHAR(255) NOT NULL DEFAULT ''     COMMENT '当前活跃节点ID(逗号分隔)',
  `status`            TINYINT      NOT NULL DEFAULT 1      COMMENT '状态:1=运行中,2=已通过,4=已撤销,5=已终止,6=已退回(待重提),7=已撤回(发起人收回,待重提)',
  `start_user_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '发起人ID',
  `start_user_name`   VARCHAR(64)  NOT NULL DEFAULT ''     COMMENT '发起人昵称(冗余)',
  `finished_at`       DATETIME     NULL                   COMMENT '结束时间',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '最后修改人ID(ORM自动填充)',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME     DEFAULT NULL            COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  KEY `idx_definition` (`definition_id`),
  KEY `idx_start_user` (`start_user_id`),
  KEY `idx_biz` (`flow_key`, `biz_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='审批流程实例';

-- ------------------------------------------------------------
-- 审批任务 (会签/或签按"节点×审批人"一人一行; 抄送为待阅行)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `wf_task` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `instance_id`   BIGINT UNSIGNED NOT NULL             COMMENT '实例ID',
  `node_id`       VARCHAR(32)  NOT NULL                COMMENT '节点ID(树内唯一)',
  `node_name`     VARCHAR(64)  NOT NULL DEFAULT ''     COMMENT '节点名称(冗余)',
  `node_type`     TINYINT      NOT NULL DEFAULT 1      COMMENT '节点类型:1=审批,2=抄送',
  `sign_type`     TINYINT      NOT NULL DEFAULT 1      COMMENT '签核方式:1=或签,2=会签',
  `assignee_id`   BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '处理人ID',
  `assignee_name` VARCHAR(64)  NOT NULL DEFAULT ''     COMMENT '处理人昵称(冗余)',
  `delegate_from_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '委派来源任务ID(0=非被委派任务,>0=被委派的代办任务)',
  `status`        TINYINT      NOT NULL DEFAULT 1      COMMENT '状态:1=待办,2=已同意,3=已驳回,4=已转出,5=已作废,6=已失效(退回/撤销后原同意失效),7=已委派,8=委办完成',
  `comment`       VARCHAR(500) NOT NULL DEFAULT ''     COMMENT '审批意见',
  `receive_time`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '到达时间',
  `due_time`      DATETIME     NULL                   COMMENT '办理期限(节点超时配置物化,NULL=不限)',
  `acted_at`      DATETIME     NULL                   COMMENT '处理时间',
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_instance` (`instance_id`),
  KEY `idx_assignee` (`assignee_id`, `status`),
  KEY `idx_status_due` (`status`, `due_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='审批任务';

-- ------------------------------------------------------------
-- 流转记录 (只追加时间线, 含系统动作; operator=0 表示系统)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `wf_record` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `instance_id`   BIGINT UNSIGNED NOT NULL             COMMENT '实例ID',
  `task_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '关联任务ID(无则为0)',
  `node_id`       VARCHAR(32)  NOT NULL DEFAULT ''     COMMENT '节点ID',
  `node_name`     VARCHAR(64)  NOT NULL DEFAULT ''     COMMENT '节点名称',
  `action`        VARCHAR(16)  NOT NULL                COMMENT '动作:submit/resubmit/approve/reject/back/cancel/withdraw/cc/finish/transfer/delegate/delegateResolve/terminate/urge/append/reduce/timeoutRemind/timeoutTransfer/timeoutApprove',
  `operator_id`   BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '操作人ID(0=系统)',
  `operator_name` VARCHAR(64)  NOT NULL DEFAULT ''     COMMENT '操作人昵称(0=系统)',
  `comment`       VARCHAR(500) NOT NULL DEFAULT ''     COMMENT '备注/意见',
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_instance` (`instance_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='审批流转记录';

-- ------------------------------------------------------------
-- 岗位管理: 审批人解析依据 (指定岗位 / 部门主管岗)
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `sys_post` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `post_code`  VARCHAR(32)  NOT NULL                COMMENT '岗位编码(唯一, 如 hr, dept_leader)',
  `post_name`  VARCHAR(64)  NOT NULL                COMMENT '岗位名称',
  `post_kind`  TINYINT      NOT NULL DEFAULT 1      COMMENT '岗位类型:1=普通岗,2=主管岗(部门主管解析依据)',
  `sort`       INT          NOT NULL DEFAULT 0      COMMENT '排序',
  `status`     TINYINT      NOT NULL DEFAULT 1      COMMENT '状态:1=启用,0=禁用',
  `remark`     VARCHAR(255) NOT NULL DEFAULT ''     COMMENT '备注',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '最后修改人ID(ORM自动填充)',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME     DEFAULT NULL            COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_post_code` (`post_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='岗位管理';

-- 用户挂岗 (谁在哪个组织担任什么岗位; 纯关联表, 物理删除, 无审计列)
CREATE TABLE IF NOT EXISTS `sys_user_post` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id`    BIGINT UNSIGNED NOT NULL                COMMENT '用户ID',
  `post_id`    BIGINT UNSIGNED NOT NULL                COMMENT '岗位ID',
  `org_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '组织ID(0=不限定组织)',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_post_org` (`user_id`, `post_id`, `org_id`),
  KEY `idx_post` (`post_id`),
  KEY `idx_org` (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户岗位关联';

-- 内置两个常用岗位 (原 p002 种子)
INSERT IGNORE INTO `sys_post` (`post_code`,`post_name`,`post_kind`,`sort`,`status`,`remark`) VALUES
  ('dept_leader', '部门主管', 2, 1, 1, '主管岗: 部门主管解析时优先取挂此岗的用户(可自定义多个主管岗)'),
  ('general_manager', '总经理', 2, 2, 1, '示例主管岗');

-- ------------------------------------------------------------
-- 菜单 (9000 号段: 审批中心 + 岗位管理; 岗位管理挂社区版"系统管理"目录 id=1 下)
-- ------------------------------------------------------------
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (9000, 0,    '审批中心', 1, '/flow',              'Layout',                 'Stamp',   'flow',                 30, 1, 1),
  (9010, 9000, '流程定义', 2, '/flow/definitions',  'flow/definitions/index', 'Tickets', 'flow:definition:list', 31, 1, 1),
  (9020, 9000, '我的审批', 2, '/flow/center',       'flow/center/index',      'Check',   'flow:instance:list',   32, 1, 1),
  -- 流程详情页: 侧边栏不可见 (从列表/待办跳转进入)
  (9030, 9000, '流程详情', 2, '/flow/detail',       'flow/detail/index',      '',        'flow:instance:list',   33, 0, 1),
  -- 实例管理 (管理员全局视角, 默认仅超管可见): 全部审批记录 + 动态加签/减签/中止
  (9050, 9000, '实例管理', 2, '/flow/instances',    'flow/instances/index',   'Files',   'flow:instance:manage', 34, 1, 1),
  -- 按钮权限
  (9011, 9010, '定义新增', 3, '', '', '', 'flow:definition:create',  1, 0, 1),
  (9012, 9010, '定义修改', 3, '', '', '', 'flow:definition:update',  2, 0, 1),
  (9013, 9010, '定义删除', 3, '', '', '', 'flow:definition:delete',  3, 0, 1),
  (9014, 9010, '定义发布', 3, '', '', '', 'flow:definition:publish', 4, 0, 1),
  (9021, 9020, '发起流程', 3, '', '', '', 'flow:instance:start',    1, 0, 1),
  (9022, 9020, '撤销流程', 3, '', '', '', 'flow:instance:cancel',   2, 0, 1),
  (9023, 9020, '审批操作', 3, '', '', '', 'flow:task:handle',       3, 0, 1),
  (9040, 1, '岗位管理', 2, '/system/posts', 'system/posts/index', 'Suitcase', 'system:post:list', 15, 1, 1),
  (9041, 9040, '岗位新增', 3, '', '', '', 'system:post:create', 1, 0, 1),
  (9042, 9040, '岗位修改', 3, '', '', '', 'system:post:update', 2, 0, 1),
  (9043, 9040, '岗位删除', 3, '', '', '', 'system:post:delete', 3, 0, 1);

-- ------------------------------------------------------------
-- API 资源 (依赖 sys_api uk_path_method 唯一键幂等)
-- ------------------------------------------------------------
INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/flow/definitions',            'GET',    '审批中心', '流程定义列表'),
  ('/api/v1/flow/definitions/usable',     'GET',    '审批中心', '可发起的流程列表(最新发布版)'),
  ('/api/v1/flow/definitions',            'POST',   '审批中心', '新增流程定义(草稿)'),
  ('/api/v1/flow/definitions/{id}',       'GET',    '审批中心', '流程定义详情'),
  ('/api/v1/flow/definitions/{id}',       'PUT',    '审批中心', '修改流程定义(仅草稿)'),
  ('/api/v1/flow/definitions/{id}',       'DELETE', '审批中心', '删除流程定义(仅草稿)'),
  ('/api/v1/flow/definitions/{id}/publish', 'POST', '审批中心', '发布流程定义(生成新版本)'),
  ('/api/v1/flow/definitions/{id}/disable', 'POST', '审批中心', '停用流程定义版本'),
  ('/api/v1/flow/designer/options',       'GET',    '审批中心', '设计器选项(用户/角色)'),
  ('/api/v1/flow/instances',              'GET',    '审批中心', '流程实例列表(待办/已办/我发起/抄送/全部-管理员)'),
  ('/api/v1/flow/instances',              'POST',   '审批中心', '发起流程'),
  ('/api/v1/flow/instances/{id}',         'GET',    '审批中心', '流程实例详情'),
  ('/api/v1/flow/instances/{id}/cancel',  'POST',   '审批中心', '撤销流程(发起人)'),
  ('/api/v1/flow/instances/{id}/withdraw', 'POST',  '审批中心', '撤回流程(发起人,尚无审批时收回待改)'),
  ('/api/v1/flow/instances/{id}/resubmit','POST',   '审批中心', '重新提交(退回后)'),
  ('/api/v1/flow/instances/{id}/terminate','POST',  '审批中心', '终止流程(管理员)'),
  ('/api/v1/flow/instances/{id}/urge',    'POST',   '审批中心', '催办(发起人, 10分钟限一次)'),
  ('/api/v1/flow/tasks/{id}/transfer',    'POST',   '审批中心', '转办(待办转给他人)'),
  ('/api/v1/flow/tasks/{id}/delegate',        'POST', '审批中心', '委派(代办后回到原审批人终审)'),
  ('/api/v1/flow/tasks/{id}/delegateResolve', 'POST', '审批中心', '委派处理(被委托人提交意见)'),
  ('/api/v1/flow/tasks/{id}/append',      'POST',   '审批中心', '加签(当前节点追加必要审批人)'),
  ('/api/v1/flow/tasks/{id}/reduce',      'POST',   '审批中心', '减签(移除节点待办审批人)'),
  ('/api/v1/flow/tasks/{id}/approve',     'POST',   '审批中心', '同意'),
  ('/api/v1/flow/tasks/{id}/reject',      'POST',   '审批中心', '驳回'),
  ('/api/v1/flow/tasks/{id}/read',        'PUT',    '审批中心', '抄送已读'),
  ('/api/v1/flow/tasks/count',            'GET',    '审批中心', '待办/待阅数量'),
  ('/api/v1/system/posts',                'GET',    '岗位管理', '岗位分页列表'),
  ('/api/v1/system/posts/all',            'GET',    '岗位管理', '启用岗位全量'),
  ('/api/v1/system/posts',                'POST',   '岗位管理', '新增岗位'),
  ('/api/v1/system/posts/{id}',           'PUT',    '岗位管理', '修改岗位'),
  ('/api/v1/system/posts/{id}',           'DELETE', '岗位管理', '删除岗位'),
  ('/api/v1/system/posts/{id}/members',   'GET',    '岗位管理', '岗位成员列表'),
  ('/api/v1/system/posts/{id}/members',   'POST',   '岗位管理', '添加岗位成员'),
  ('/api/v1/system/posts/members/{relId}','DELETE', '岗位管理', '移除岗位成员'),
  -- 社区版模块历史缺漏补种 (原 p013, tools/genapi -check 扫描发现)
  ('/api/v1/auth/avatar',                'POST', '认证',     '上传头像'),
  ('/api/v1/auth/login/totp',            'POST', '认证',     '两步验证登录'),
  ('/api/v1/auth/public-key',            'GET',  '认证',     '获取登录加密公钥'),
  ('/api/v1/auth/totp/disable',          'PUT',  '认证',     '解绑两步验证'),
  ('/api/v1/auth/totp/enable',           'PUT',  '认证',     '绑定两步验证'),
  ('/api/v1/auth/totp/setup',            'GET',  '认证',     '生成两步验证密钥'),
  ('/api/v1/message/events',             'GET',  '消息通知', '消息事件流(SSE)'),
  ('/api/v1/system/apis/all',            'GET',  'API管理',  '全量API'),
  ('/api/v1/system/roles/all',           'GET',  '角色管理', '全量角色'),
  ('/api/v1/system/users/{id}/password', 'PUT',  '用户管理', '重置密码');

-- ############################################################
-- 二、AI 智能对话 (原 p003/p005/p006/p007)
-- OpenAI 兼容接口 (基于 langchaingo), 配置存 sys_config (全局配置页维护);
-- 会话与消息 MySQL 持久化, 上下文取最近 20 条, 历史查看全量。
-- ############################################################

-- 会话与消息表 (原 p007)
CREATE TABLE IF NOT EXISTS `ai_conversation` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `session_id`    VARCHAR(64)  NOT NULL                   COMMENT '会话ID(前端生成并保管)',
  `user_id`       BIGINT UNSIGNED NOT NULL                COMMENT '所属用户ID',
  `title`         VARCHAR(128) NOT NULL DEFAULT ''        COMMENT '会话标题(首条用户消息裁剪)',
  `message_count` INT UNSIGNED NOT NULL DEFAULT 0         COMMENT '累计消息条数(user+assistant)',
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最近一轮对话时间',
  `deleted_at`    DATETIME     DEFAULT NULL               COMMENT '未使用(保留列对齐代码生成器约定)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_session_id` (`session_id`),
  KEY `idx_user_updated` (`user_id`,`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI会话';

CREATE TABLE IF NOT EXISTS `ai_chat_message` (
  `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `conversation_id` BIGINT UNSIGNED NOT NULL               COMMENT '会话ID(ai_conversation.id)',
  `role`            VARCHAR(16)  NOT NULL                  COMMENT '角色:user/assistant/tool(tool=工具调用步骤,内容为步骤JSON)',
  `content`         MEDIUMTEXT   NOT NULL                  COMMENT '消息正文(思考过程不落库)',
  `prompt_tokens`     INT UNSIGNED DEFAULT NULL            COMMENT '输入token用量(仅assistant行,本轮累计)',
  `completion_tokens` INT UNSIGNED DEFAULT NULL            COMMENT '输出token用量(仅assistant行,本轮累计)',
  `total_tokens`      INT UNSIGNED DEFAULT NULL            COMMENT '总token用量(仅assistant行;网关未回报为0)',
  `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at`      DATETIME     DEFAULT NULL               COMMENT '未使用(保留列对齐代码生成器约定)',
  PRIMARY KEY (`id`),
  KEY `idx_conversation` (`conversation_id`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI会话消息';

-- 菜单 (9100 号段: AI 助手)
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (9100, 0,    'AI 助手', 1, '/ai',       'Layout',            'MagicStick',   'ai',            40, 1, 1),
  (9110, 9100, '智能对话', 2, '/ai/chat', 'ai/chat/index',     'ChatDotRound', 'ai:chat:list',  41, 1, 1);

-- API 资源 (原 p003/p005/p006)
INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/ai/chat',     'POST',   'AI助手', 'AI对话(流式SSE)'),
  ('/api/v1/ai/config',   'GET',    'AI助手', 'AI配置状态'),
  ('/api/v1/ai/history',  'GET',    'AI助手', 'AI会话历史(恢复界面)'),
  ('/api/v1/ai/sessions', 'GET',    'AI助手', 'AI会话列表(历史会话栏)'),
  ('/api/v1/ai/sessions', 'DELETE', 'AI助手', 'AI会话删除');

-- 配置种子 (api_key 不预置, 管理员自行填写; uk_config_key 幂等)
INSERT IGNORE INTO `sys_config` (`config_key`, `config_value`, `config_type`, `name`, `remark`, `sort`) VALUES
  ('ai.base_url',      'https://api.openai.com/v1', 0, 'AI接口地址',  'OpenAI 兼容地址, 可换 DeepSeek/通义兼容模式/Ollama 等', 50),
  ('ai.model',         'gpt-4o-mini',               0, 'AI模型',      '如 gpt-4o-mini / deepseek-chat / qwen-plus',           51),
  ('ai.temperature',   '0.7',                       1, 'AI温度',      '0-2, 越低越确定, 越高越发散',                          52),
  ('ai.system_prompt', '',                          0, 'AI系统提示词', '全局系统提示词, 可空',                                 53),
  ('ai.api_key',       '',                          0, 'AI API Key',  'OpenAI 兼容接口密钥, 填写后 AI 对话可用',               54);

-- ############################################################
-- 三、微信公众号对接 (原 p004, 占位, 基于 silenceper/wechat/v2)
-- 回调地址: /wechat/callback (根路由, 免鉴权, 供微信服务器调用;
--          部署时需暴露公网, nginx 反代规则同 /upload)
-- 消息处理: Echo 占位 (回复用户发送的文本)
-- ############################################################

INSERT IGNORE INTO `sys_config` (`config_key`, `config_value`, `config_type`, `name`, `remark`, `sort`) VALUES
  ('wechat.app_id',           '', 0, '微信公众号AppID',     '公众号开发信息中的 AppID',                 60),
  ('wechat.app_secret',       '', 0, '微信公众号AppSecret', '公众号开发信息中的 AppSecret',             61),
  ('wechat.token',            '', 0, '微信服务器Token',      '公众号服务器配置的 Token (用于URL验签)',   62),
  ('wechat.encoding_aes_key', '', 0, '消息加解密密钥',       'EncodingAESKey, 明文模式可留空',           63);

-- ############################################################
-- 四、业务审批 Demo: 请假申请 (原 p016)
-- 演示「业务表 + 编程式接入」: 业务数据存 biz_leave 单表, 不走流程表单设计器;
-- 后端 leave 模块调 flow.StartForBiz(flowKey="biz_leave") 发起, flow_status 全部由
-- flow.RegisterBizListener 四回调 (通过/退回/撤销/终止) 写回 —— 业务侧不写审批状态机。
-- 演示流程定义 (条件分支+会签+抄送) 与测试账号授权依赖 p012, 见
-- upgrade-modules/p016_biz_leave_demo.sql (可选种子)。
-- ############################################################

CREATE TABLE IF NOT EXISTS `biz_leave` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `leave_type`    TINYINT      NOT NULL DEFAULT 1         COMMENT '请假类型:1=事假,2=病假,3=年假,4=调休,5=其他',
  `start_date`    DATE         NOT NULL                   COMMENT '开始日期',
  `end_date`      DATE         NOT NULL                   COMMENT '结束日期',
  `days`          DECIMAL(5,1) NOT NULL DEFAULT 1.0       COMMENT '请假天数(0.5天粒度,申请人填报)',
  `reason`        VARCHAR(500) NOT NULL DEFAULT ''        COMMENT '请假事由',
  `flow_status`   TINYINT      NOT NULL DEFAULT 0         COMMENT '审批状态:0=审批中,1=已通过,2=被退回,3=已撤销,4=已终止(引擎回调写入;未发起时无意义)',
  `flow_instance` BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '流程实例ID(0=未发起/草稿)',
  `create_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '创建人ID(ORM自动填充)',
  `update_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0   COMMENT '最后修改人ID(ORM自动填充)',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME     DEFAULT NULL            COMMENT '删除时间(软删)',
  PRIMARY KEY (`id`),
  KEY `idx_create` (`create_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='请假申请(业务审批Demo)';

-- 菜单 (9200 号段: 业务审批)
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (9200, 0,    '业务审批', 1, '/biz',       'Layout',           'Calendar', 'biz',              35, 1, 1),
  (9210, 9200, '请假申请', 2, '/biz/leave', 'biz/leave/index',  'AlarmClock', 'biz:leave:list', 36, 1, 1),
  (9211, 9210, '假单新增', 3, '', '', '', 'biz:leave:create', 1, 0, 1),
  (9212, 9210, '假单修改', 3, '', '', '', 'biz:leave:update', 2, 0, 1),
  (9213, 9210, '假单删除', 3, '', '', '', 'biz:leave:delete', 3, 0, 1),
  (9214, 9210, '提交审批', 3, '', '', '', 'biz:leave:submit', 4, 0, 1),
  (9215, 9210, '撤销审批', 3, '', '', '', 'biz:leave:cancel', 5, 0, 1);

-- API 资源
INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/leaves',              'GET',    '业务审批', '请假申请列表(本人;admin全部)'),
  ('/api/v1/leaves',              'POST',   '业务审批', '新增请假申请(草稿)'),
  ('/api/v1/leaves/{id}',         'PUT',    '业务审批', '修改请假申请(未在审批流中)'),
  ('/api/v1/leaves/{id}',         'DELETE', '业务审批', '删除请假申请(未在审批流中)'),
  ('/api/v1/leaves/{id}/submit',  'POST',   '业务审批', '提交审批(草稿发起/退回撤销后重提)'),
  ('/api/v1/leaves/{id}/cancel',  'POST',   '业务审批', '撤销审批(发起人,运行中或退回态)');

-- ############################################################
-- 五、流程超时扫描定时任务 (原 p018)
-- 审批节点配置办理期限后, 引擎生成任务时物化 due_time;
-- 本任务按节点超时策略处理逾期: 提醒/自动转办/自动通过。
-- 幂等: handler 未种子过才插入; 网页端 定时任务 页可调频率/暂停。
-- ############################################################
INSERT IGNORE INTO `sys_job` (`name`,`handler`,`cron_expr`,`params`,`status`,`remark`)
SELECT '流程超时扫描', 'flow.timeoutScan', '0 */10 * * * *', '', 1,
       '审批任务超时处理: 提醒/自动转办/自动通过 (按审批节点超时配置, 无配置不受影响)'
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `sys_job` WHERE `handler` = 'flow.timeoutScan' AND `deleted_at` IS NULL);

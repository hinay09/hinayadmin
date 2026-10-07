-- ============================================================
-- modules_init: 模块全量种子 (由 p001~p014 合并, 2026-10-06; p012 演示种子独立保留)
--
-- 执行顺序: 先执行社区版基础库 manifest/sql/init.sql, 再执行本文件。
-- 幂等: 建表 IF NOT EXISTS / 种子 INSERT IGNORE, 可重复执行。
--
-- 模块:
--   一、自由审批流 + 岗位管理   (原 p001/p002)
--   二、AI 智能对话             (原 p003/p005/p006/p007)
--   三、微信公众号对接          (原 p004)
--
-- 合并说明:
--   原 p002 的存量迁移逻辑 (wf_instance ALTER 加列/回填快照) 不再保留 ——
--   p001 的建表语句已含 v2 全部列, 本文件面向全新部署; 保留 p002 独有的内置岗位种子。
--   AI 会话持久化 (原 p007): 历史记录与上下文记忆存 MySQL, 旧 Redis key
--   (ai:chat:sess:* / ai:chat:sessidx:*) 不迁移, 随 TTL 最长 7 天自然过期;
--   ai 两表无 create_id/update_id 审计列 (行为人即 user_id), 不纳入
--   ormfill fillTables 白名单 (同 sys_user_totp / wf_task / wf_record);
--   会话删除走物理删除 (Unscoped): uk_session_id 与软删残留行冲突 (同 0011 先例)。
--
-- 2026-10-06 二次合并 (面向全新部署, upgrade-modules 仅余本文件与 p012):
--   p008/p009/p010 的列与注释 (wf_instance.self_selects、wf_task.status 6=已失效、
--   wf_record.action 新动作) 已并入建表 DDL, 不再保留独立脚本;
--   p011 (flow 后增 5 条) 与 p013 (社区版历史缺漏 10 条) 已并入 sys_api 种子块;
--   p014 操作人显示名回填仅面向存量库 (新装无旧数据, 且 Auth 中间件已逐请求
--   回填昵称), 不保留;
--   p012 流程演示/测试种子为可选脚本, 独立保留 (p012_flow_demo_test_seed.sql)。
--
-- 2026-10-06 三次增量 (p015): AI 工具调用第一层增强 —— 工具步骤落库 (ai_chat_message
--   新增 role=tool 行, 内容为步骤 JSON) + token 用量列 (prompt/completion/total, 仅
--   assistant 行), 已并入下方 ai_chat_message 建表 DDL; 存量库执行 p015 增量脚本。
--
-- 2026-10-06 四次增量 (p016): 业务型流程审批 Demo —— 请假申请 (biz_leave 单表),
--   编程式接入审批流 (StartForBiz 发起 + BizListener 回调写 flow_status, 不走
--   表单设计器), 已并入下方 "四、业务审批 Demo" 段; 演示流程定义/测试账号授权
--   依赖 p012, 由 p016_biz_leave_demo.sql 独立提供。
--
-- 2026-10-06 五次增量 (p017): 流程实例管理 —— 实例列表新增管理员 all 视角
--   (代码层新增 scope, 无表结构变化), 菜单 9050 与 sys_api 描述已并入下方种子块;
--   存量库执行 p017_flow_instance_manage.sql。
-- ============================================================

-- ############################################################
-- 一、自由审批流 + 岗位管理 (原 p001/p002)
-- 模型: 单行定义(发布原地生效) + 实例携带表单/节点树快照(在途不受定义后续修改影响)
--       + 任务按"节点×审批人"落行; 岗位管理为审批"指定岗位"与"部门主管"解析依据。
-- 说明: wf_task / wf_record 为行为表, 不建审计列, 不纳入 ormfill;
--       wf_definition / wf_instance / sys_post 已在 ormfill 登记; sys_user_post 纯关联表豁免。
-- 依赖: init.sql (sys_menu/sys_api); 菜单使用 9000 号段。
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
  `status`            TINYINT      NOT NULL DEFAULT 1      COMMENT '状态:1=运行中,2=已通过,4=已撤销,5=已终止,6=已退回(待重提)',
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
  `status`        TINYINT      NOT NULL DEFAULT 1      COMMENT '状态:1=待办,2=已同意,3=已驳回,4=已转出,5=已作废,6=已失效(退回/撤销后原同意失效)',
  `comment`       VARCHAR(500) NOT NULL DEFAULT ''     COMMENT '审批意见',
  `receive_time`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '到达时间',
  `acted_at`      DATETIME     NULL                   COMMENT '处理时间',
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_instance` (`instance_id`),
  KEY `idx_assignee` (`assignee_id`, `status`)
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
  `action`        VARCHAR(16)  NOT NULL                COMMENT '动作:submit/resubmit/approve/reject/back/cancel/cc/finish/transfer/terminate/urge/append/reduce',
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
  ('/api/v1/flow/instances/{id}/resubmit','POST',   '审批中心', '重新提交(退回后)'),
  ('/api/v1/flow/instances/{id}/terminate','POST',  '审批中心', '终止流程(管理员)'),
  ('/api/v1/flow/instances/{id}/urge',    'POST',   '审批中心', '催办(发起人, 10分钟限一次)'),
  ('/api/v1/flow/tasks/{id}/transfer',    'POST',   '审批中心', '转办(待办转给他人)'),
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

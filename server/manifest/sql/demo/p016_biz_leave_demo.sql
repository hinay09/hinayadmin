-- ============================================================
-- p016: 业务型流程审批 Demo —— 请假申请 (biz_leave)
--
-- 演示「业务表 + 编程式接入」模式 (docs/pro/flow-integration.md):
--   业务数据存自己的表 biz_leave, 不走流程表单设计器;
--   后端 leave 模块调 flow.StartForBiz 发起, flow_status 全部由
--   flow.RegisterBizListener 四回调 (通过/退回/撤销/终止) 写回。
--
-- 内容:
--   1) biz_leave 建表 (含 flow_status/flow_instance 冗余列, 审计列齐全)
--   2) 菜单 9200 号段: 业务审批目录 + 请假申请页 + 按钮权限
--   3) sys_api 种子 (leave 模块 6 个接口)
--   4) 演示流程定义 wf_definition (id=910004, flow_key=biz_leave):
--        张三发起 → 李主管或签 → 天数>3 加赵总监会签 → 抄送钱同事
--        (审批人/抄送人为 p012 测试账号 910002/910004/910005)
--   5) Casbin: 流程测试角色(910001) 增补菜单/API 授权
--
-- 依赖: init.sql (含 biz_leave 表/菜单/API) + p012_flow_demo_test_seed.sql (审批人账号);
--       本脚本提供其独有内容: 演示流程定义(910004) + Casbin 授权。
-- 幂等: 建表 IF NOT EXISTS / 种子 INSERT IGNORE, 可重复执行。
-- 注意: 直改 casbin_rule 后需重启服务 (或在角色管理里重新保存任一角色) 触发策略重载。
-- 清理语句见文件尾部 (默认注释)。
-- ============================================================

-- ------------------------------------------------------------
-- 1. 请假申请业务表 (单表; 流程状态列由引擎回调写入)
-- ------------------------------------------------------------
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

-- ------------------------------------------------------------
-- 2. 菜单 (9200 号段: 业务审批)
-- ------------------------------------------------------------
INSERT IGNORE INTO `sys_menu` (`id`,`parent_id`,`name`,`type`,`path`,`component`,`icon`,`permission`,`sort`,`visible`,`status`) VALUES
  (9200, 0,    '业务审批', 1, '/biz',       'Layout',           'Calendar', 'biz',              35, 1, 1),
  (9210, 9200, '请假申请', 2, '/biz/leave', 'biz/leave/index',  'AlarmClock', 'biz:leave:list', 36, 1, 1),
  -- 按钮权限
  (9211, 9210, '假单新增', 3, '', '', '', 'biz:leave:create', 1, 0, 1),
  (9212, 9210, '假单修改', 3, '', '', '', 'biz:leave:update', 2, 0, 1),
  (9213, 9210, '假单删除', 3, '', '', '', 'biz:leave:delete', 3, 0, 1),
  (9214, 9210, '提交审批', 3, '', '', '', 'biz:leave:submit', 4, 0, 1),
  (9215, 9210, '撤销审批', 3, '', '', '', 'biz:leave:cancel', 5, 0, 1);

-- ------------------------------------------------------------
-- 3. API 资源 (依赖 sys_api uk_path_method 唯一键幂等)
-- ------------------------------------------------------------
INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/leaves',              'GET',    '业务审批', '请假申请列表(本人;admin全部)'),
  ('/api/v1/leaves',              'POST',   '业务审批', '新增请假申请(草稿)'),
  ('/api/v1/leaves/{id}',         'PUT',    '业务审批', '修改请假申请(未在审批流中)'),
  ('/api/v1/leaves/{id}',         'DELETE', '业务审批', '删除请假申请(未在审批流中)'),
  ('/api/v1/leaves/{id}/submit',  'POST',   '业务审批', '提交审批(草稿发起/退回撤销后重提)'),
  ('/api/v1/leaves/{id}/cancel',  'POST',   '业务审批', '撤销审批(发起人,运行中或退回态)');

-- ------------------------------------------------------------
-- 4. 演示流程定义 (已发布 v1; 依赖 p012 测试账号)
--    天数>3: 李主管或签 → 赵总监会签 → 抄送钱同事
--    3天及内: 李主管或签 → 抄送钱同事
--    form_conf 仅作审批详情页只读快照展示 + 条件分支取 days;
--    业务数据本体在 biz_leave 表, 不走表单设计器维护。
-- ------------------------------------------------------------
INSERT IGNORE INTO `wf_definition` (`id`,`flow_key`,`name`,`form_conf`,`flow_conf`,`version`,`status`,`remark`,`create_id`) VALUES
  (910004, 'biz_leave', '请假审批(业务Demo)',
   '[{"key":"leave_type","label":"请假类型","type":"select","options":["事假","病假","年假","调休","其他"]},{"key":"date_range","label":"请假时间","type":"input"},{"key":"days","label":"请假天数","type":"number","required":true},{"key":"reason","label":"请假事由","type":"textarea","required":true}]',
   '{"id":"start","type":"start","name":"发起人","child":{"id":"n1","type":"approver","name":"直属主管审批","approverType":"user","approverIds":[910002],"signType":"any","child":{"id":"n2","type":"condition","name":"天数判断","branches":[{"id":"b1","name":"超过3天 (加总监会签)","isDefault":false,"conditions":[[{"field":"days","op":"gt","value":3}]],"child":{"id":"n3","type":"approver","name":"总监会签","approverType":"user","approverIds":[910004],"signType":"all","child":{"id":"n4","type":"cc","name":"抄送知会","approverType":"user","approverIds":[910005]}}},{"id":"b2","name":"3天及以内","isDefault":true,"child":{"id":"n5","type":"cc","name":"抄送知会","approverType":"user","approverIds":[910005]}}]}}}',
   1, 1, '【业务审批Demo】p016 种子流程: 业务表 biz_leave 编程式接入 (StartForBiz + 回调写 flow_status); 审批人依赖 p012 测试账号, 可随时删除', 1);

-- ------------------------------------------------------------
-- 5. Casbin 授权: 流程测试角色 (p012, role id=910001) 增补请假菜单与 API
-- ------------------------------------------------------------
INSERT IGNORE INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  ('p', '910001', 'menu:9200', 'access', '', '', ''),
  ('p', '910001', 'menu:9210', 'access', '', '', ''),
  ('p', '910001', 'menu:9211', 'access', '', '', ''),
  ('p', '910001', 'menu:9212', 'access', '', '', ''),
  ('p', '910001', 'menu:9213', 'access', '', '', ''),
  ('p', '910001', 'menu:9214', 'access', '', '', ''),
  ('p', '910001', 'menu:9215', 'access', '', '', ''),
  ('p', '910001', '/api/v1/leaves',             'GET',    '', '', ''),
  ('p', '910001', '/api/v1/leaves',             'POST',   '', '', ''),
  ('p', '910001', '/api/v1/leaves/{id}',        'PUT',    '', '', ''),
  ('p', '910001', '/api/v1/leaves/{id}',        'DELETE', '', '', ''),
  ('p', '910001', '/api/v1/leaves/{id}/submit', 'POST',   '', '', ''),
  ('p', '910001', '/api/v1/leaves/{id}/cancel', 'POST',   '', '', '');

-- ============================================================
-- 清理 (演练完成后手动执行; 依赖顺序: 定义 → 授权 → 菜单/API → 业务表)
-- ============================================================
-- DELETE FROM `wf_definition` WHERE `id` = 910004;
-- DELETE FROM `casbin_rule` WHERE (`ptype`='p' AND `v0`='910001' AND (`v1` LIKE 'menu:92%' OR `v1` LIKE '/api/v1/leaves%'));
-- DELETE FROM `sys_menu` WHERE `id` BETWEEN 9200 AND 9215;
-- DELETE FROM `sys_api`  WHERE `path` LIKE '/api/v1/leaves%';
-- DELETE FROM `biz_leave` WHERE 1=1;
-- 执行后同样需重启服务 (或重新保存任一角色) 重载 Casbin。

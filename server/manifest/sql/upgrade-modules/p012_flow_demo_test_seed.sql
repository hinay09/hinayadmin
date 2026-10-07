-- ============================================================
-- p012: 流程演示与测试数据 (可选种子脚本 —— 生产库可不执行!)
--
-- 内容:
--   1) 5 个流程测试账号 (910001-910005) + 1 个流程测试角色 (910001, code=flow_demo)
--   2) 测试角色的最小授权: 仪表盘/个人中心/消息中心 + 审批中心(发起/审批/撤销)
--      与全部审批任务 API (含加签/减签/转办), 不含流程定义管理
--   3) 3 条已发布的演示流程定义 (910001-910005 用户为固定审批人):
--        test_leave      【测试】请假演示      — 或签 + 抄送
--        test_expense    【测试】报销演示      — 条件分支(金额>1000) + 财务/总监会签
--        test_selfselect 【测试】自选审批演示   — 发起人自选审批人
--
-- 账号 (统一密码 Flow@Test2026, must_change_pwd=0 可直接登录):
--   flow_demo_zhang 张三(常用发起人) / flow_demo_li 李主管 / flow_demo_wang 王财务
--   flow_demo_zhao 赵总监 / flow_demo_qian 钱同事(抄送/转办对象)
--
-- 号段: 固定主键 910000 段, 与业务自增 ID 错开; 依赖各表唯一键 INSERT IGNORE 幂等, 可重复执行。
-- 注意: ① 直改 casbin_rule 后需重启服务 (或在角色管理里重新保存任一角色) 触发策略重载才生效;
--       ② 依赖 init.sql (菜单/基础表) 与 modules_init.sql (审批中心菜单 9000 段);
--       ③ 演练产生的实例/任务数据清理语句见文件尾部 (默认注释)。
-- ============================================================

-- ------------------------------------------------------------
-- 1. 测试角色 (data_scope=5 仅本人)
-- ------------------------------------------------------------
INSERT IGNORE INTO `sys_role` (`id`,`name`,`code`,`sort`,`status`,`remark`,`data_scope`) VALUES
  (910001, '流程测试角色', 'flow_demo', 90, 1, '【流程演示与测试】p012 种子角色: 审批中心最小权限, 可随时删除', 5);

-- ------------------------------------------------------------
-- 2. 测试账号 (密码均为 Flow@Test2026 的 bcrypt)
-- ------------------------------------------------------------
INSERT IGNORE INTO `sys_user` (`id`,`username`,`password`,`nickname`,`status`,`must_change_pwd`,`remark`) VALUES
  (910001, 'flow_demo_zhang', '$2b$10$AvhHEh6O7UlDNEBKpTs9/u2XS/dTL669hoy5EjxZfY7VVPBU/387u', '流程测试·张三',   1, 0, '【流程演示与测试】p012 种子账号 (常用发起人), 密码 Flow@Test2026'),
  (910002, 'flow_demo_li',    '$2b$10$AvhHEh6O7UlDNEBKpTs9/u2XS/dTL669hoy5EjxZfY7VVPBU/387u', '流程测试·李主管', 1, 0, '【流程演示与测试】p012 种子账号 (直属/部门主管), 密码 Flow@Test2026'),
  (910003, 'flow_demo_wang',  '$2b$10$AvhHEh6O7UlDNEBKpTs9/u2XS/dTL669hoy5EjxZfY7VVPBU/387u', '流程测试·王财务', 1, 0, '【流程演示与测试】p012 种子账号 (财务), 密码 Flow@Test2026'),
  (910004, 'flow_demo_zhao',  '$2b$10$AvhHEh6O7UlDNEBKpTs9/u2XS/dTL669hoy5EjxZfY7VVPBU/387u', '流程测试·赵总监', 1, 0, '【流程演示与测试】p012 种子账号 (总监, 会签用), 密码 Flow@Test2026'),
  (910005, 'flow_demo_qian',  '$2b$10$AvhHEh6O7UlDNEBKpTs9/u2XS/dTL669hoy5EjxZfY7VVPBU/387u', '流程测试·钱同事', 1, 0, '【流程演示与测试】p012 种子账号 (抄送/转办对象), 密码 Flow@Test2026');

-- ------------------------------------------------------------
-- 3. Casbin 授权: 用户→角色 (g) + 角色→菜单/API (p)
--    依赖 init.sql 的基础菜单 (30/40/50/52/521/522) 与 modules_init.sql 的审批中心菜单 (9000 段)
-- ------------------------------------------------------------
INSERT IGNORE INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`) VALUES
  -- g: 五个测试账号加入流程测试角色
  ('g', '910001', '910001', '', '', '', ''),
  ('g', '910002', '910001', '', '', '', ''),
  ('g', '910003', '910001', '', '', '', ''),
  ('g', '910004', '910001', '', '', '', ''),
  ('g', '910005', '910001', '', '', '', ''),
  -- p: 菜单 —— 仪表盘/个人中心/消息中心 (与普通角色一致) + 审批中心(目录/我的审批/详情/发起/撤销/审批操作按钮)
  ('p', '910001', 'menu:30',   'access', '', '', ''),
  ('p', '910001', 'menu:40',   'access', '', '', ''),
  ('p', '910001', 'menu:50',   'access', '', '', ''),
  ('p', '910001', 'menu:52',   'access', '', '', ''),
  ('p', '910001', 'menu:521',  'access', '', '', ''),
  ('p', '910001', 'menu:522',  'access', '', '', ''),
  ('p', '910001', 'menu:9000', 'access', '', '', ''),
  ('p', '910001', 'menu:9020', 'access', '', '', ''),
  ('p', '910001', 'menu:9030', 'access', '', '', ''),
  ('p', '910001', 'menu:9021', 'access', '', '', ''),
  ('p', '910001', 'menu:9022', 'access', '', '', ''),
  ('p', '910001', 'menu:9023', 'access', '', '', ''),
  -- p: API —— 登录态/消息/选人下拉 (与普通角色一致)
  ('p', '910001', '/api/v1/auth/*',            '*',      '', '', ''),
  ('p', '910001', '/api/v1/message/inbox',     'GET',    '', '', ''),
  ('p', '910001', '/api/v1/message/inbox/*',   'GET',    '', '', ''),
  ('p', '910001', '/api/v1/message/inbox/*',   'DELETE', '', '', ''),
  ('p', '910001', '/api/v1/message/unread-count', 'GET', '', '', ''),
  ('p', '910001', '/api/v1/message/read-all',  'PUT',    '', '', ''),
  ('p', '910001', '/api/v1/message/*/read',    'PUT',    '', '', ''),
  ('p', '910001', '/api/v1/message/private',   'POST',   '', '', ''),
  ('p', '910001', '/api/v1/message/events',    'GET',    '', '', ''),
  ('p', '910001', '/api/v1/system/users',      'GET',    '', '', ''),
  -- p: API —— 审批中心 (发起/列表/详情/撤销/重提/催办)
  ('p', '910001', '/api/v1/flow/definitions/usable',  'GET',  '', '', ''),
  ('p', '910001', '/api/v1/flow/designer/options',    'GET',  '', '', ''),
  ('p', '910001', '/api/v1/flow/instances',           'GET',  '', '', ''),
  ('p', '910001', '/api/v1/flow/instances',           'POST', '', '', ''),
  ('p', '910001', '/api/v1/flow/instances/{id}',      'GET',  '', '', ''),
  ('p', '910001', '/api/v1/flow/instances/{id}/cancel',    'POST', '', '', ''),
  ('p', '910001', '/api/v1/flow/instances/{id}/resubmit',  'POST', '', '', ''),
  ('p', '910001', '/api/v1/flow/instances/{id}/urge',      'POST', '', '', ''),
  -- p: API —— 审批任务 (同意/驳回/转办/加签/减签/抄送已读/数量)
  ('p', '910001', '/api/v1/flow/tasks/{id}/approve',  'POST', '', '', ''),
  ('p', '910001', '/api/v1/flow/tasks/{id}/reject',   'POST', '', '', ''),
  ('p', '910001', '/api/v1/flow/tasks/{id}/transfer', 'POST', '', '', ''),
  ('p', '910001', '/api/v1/flow/tasks/{id}/append',   'POST', '', '', ''),
  ('p', '910001', '/api/v1/flow/tasks/{id}/reduce',   'POST', '', '', ''),
  ('p', '910001', '/api/v1/flow/tasks/{id}/read',     'PUT',  '', '', ''),
  ('p', '910001', '/api/v1/flow/tasks/count',         'GET',  '', '', '');

-- ------------------------------------------------------------
-- 4. 演示流程定义 (已发布 v1; 审批人为上面的固定测试账号)
-- ------------------------------------------------------------
INSERT IGNORE INTO `wf_definition` (`id`,`flow_key`,`name`,`form_conf`,`flow_conf`,`version`,`status`,`remark`,`create_id`) VALUES
  -- 4.1 【测试】请假演示: 发起人 → 李主管或签 → 抄送钱同事
  (910001, 'test_leave', '【测试】请假演示',
   '[{"key":"days","label":"请假天数","type":"number","required":true},{"key":"reason","label":"请假事由","type":"textarea","required":true}]',
   '{"id":"start","type":"start","name":"发起人","child":{"id":"n1","type":"approver","name":"直属主管审批","approverType":"user","approverIds":[910002],"signType":"any","child":{"id":"n2","type":"cc","name":"抄送知会","approverType":"user","approverIds":[910005]}}}',
   1, 1, '【流程演示与测试】p012 种子流程: 或签+抄送; 发起人 flow_demo_zhang, 审批 flow_demo_li, 抄送 flow_demo_qian; 可随时删除', 1),
  -- 4.2 【测试】报销演示: 发起人 → 李主管或签 → 金额>1000 走 王财务+赵总监会签, 否则仅王财务或签
  (910002, 'test_expense', '【测试】报销演示',
   '[{"key":"amount","label":"报销金额(元)","type":"number","required":true},{"key":"reason","label":"报销事由","type":"textarea","required":true}]',
   '{"id":"start","type":"start","name":"发起人","child":{"id":"n1","type":"approver","name":"部门主管审批","approverType":"user","approverIds":[910002],"signType":"any","child":{"id":"n2","type":"condition","name":"金额判断","branches":[{"id":"b1","name":"金额大于1000 (财务+总监会签)","isDefault":false,"conditions":[[{"field":"amount","op":"gt","value":1000}]],"child":{"id":"n3","type":"approver","name":"财务与总监会签","approverType":"user","approverIds":[910003,910004],"signType":"all"}},{"id":"b2","name":"小额 (仅财务审批)","isDefault":true,"child":{"id":"n4","type":"approver","name":"财务审批","approverType":"user","approverIds":[910003],"signType":"any"}}]}}}',
   1, 1, '【流程演示与测试】p012 种子流程: 条件分支+会签 (amount>1000 → 王财务+赵总监会签); 演练加签/减签建议在此流程操作; 可随时删除', 1),
  -- 4.3 【测试】自选审批演示: 发起人 → 审批人由发起时自选 (或签)
  (910003, 'test_selfselect', '【测试】自选审批演示',
   '[{"key":"reason","label":"申请事由","type":"textarea","required":true}]',
   '{"id":"start","type":"start","name":"发起人","child":{"id":"n1","type":"approver","name":"审批人(发起时自选)","approverType":"selfSelect","signType":"any"}}',
   1, 1, '【流程演示与测试】p012 种子流程: 发起人自选审批人 (发起弹窗选人); 可随时删除', 1);

-- ============================================================
-- 清理 (演练完成后手动执行; 按依赖顺序: 记录/任务 → 实例 → 定义 → 授权 → 账号/角色)
-- ============================================================
-- DELETE FROM `wf_record`     WHERE instance_id IN (SELECT id FROM `wf_instance` WHERE `flow_key` LIKE 'test\_%');
-- DELETE FROM `wf_task`       WHERE instance_id IN (SELECT id FROM `wf_instance` WHERE `flow_key` LIKE 'test\_%');
-- DELETE FROM `wf_instance`   WHERE `flow_key` LIKE 'test\_%';
-- DELETE FROM `wf_definition` WHERE `id` IN (910001, 910002, 910003);
-- DELETE FROM `casbin_rule`   WHERE (`ptype`='g' AND `v0` IN ('910001','910002','910003','910004','910005') AND `v1`='910001')
--                              OR (`ptype`='p' AND `v0`='910001');
-- DELETE FROM `sys_user`      WHERE `id` BETWEEN 910001 AND 910005;
-- DELETE FROM `sys_role`      WHERE `id` = 910001;
-- 执行后同样需重启服务 (或重新保存任一角色) 重载 Casbin。

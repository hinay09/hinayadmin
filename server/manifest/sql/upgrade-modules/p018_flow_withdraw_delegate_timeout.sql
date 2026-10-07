-- ============================================================
-- p018: 流程引擎补齐 — 发起人撤回 / 任务委派 / 审批超时
--
-- 1) 撤回: 实例状态新增 7=已撤回 (尚无审批人同意时收回, 待修改重提;
--    区别于撤销终态 4, 触发 OnWithdrawn 业务回调, 未注册时回退 OnReturned);
-- 2) 委派: wf_task 新增 delegate_from_id 溯源列 + 任务状态 7=已委派/8=委办完成
--    (原审批人挂起等待, 被委托人提交意见后回到原审批人终审; 转办是换人, 委派是代办);
-- 3) 超时: wf_task 新增 due_time 期限列 (approver 节点 timeoutHours/timeoutAction/
--    timeoutTransfer 配置在引擎生成任务时物化), 定时任务 flow.timeoutScan 扫描逾期:
--    remind=逾期提醒(每任务一次) / transfer=自动转办(转办后不再计时) / approve=自动通过。
--
-- 依赖: init.sql + modules_init.sql (wf_* 表与 9000 菜单号段)。
-- 幂等: ALTER 前判列/索引存在 (information_schema + PREPARE), 种子 INSERT 均可重复执行。
-- ============================================================

-- ------------------------------------------------------------
-- 1) wf_task 新列: delegate_from_id / due_time (存量库)
-- ------------------------------------------------------------
SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'wf_task' AND COLUMN_NAME = 'delegate_from_id') = 0,
  'ALTER TABLE `wf_task` ADD COLUMN `delegate_from_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT ''委派来源任务ID(0=非被委派任务,>0=被委派的代办任务)'' AFTER `assignee_name`',
  'SELECT 1');
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'wf_task' AND COLUMN_NAME = 'due_time') = 0,
  'ALTER TABLE `wf_task` ADD COLUMN `due_time` DATETIME NULL COMMENT ''办理期限(节点超时配置物化,NULL=不限)'' AFTER `receive_time`',
  'SELECT 1');
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

-- 超时扫描索引: status+due_time (扫 逾期待办 一条索引覆盖)
SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'wf_task' AND INDEX_NAME = 'idx_status_due') = 0,
  'ALTER TABLE `wf_task` ADD INDEX `idx_status_due` (`status`, `due_time`)',
  'SELECT 1');
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

-- ------------------------------------------------------------
-- 2) 状态/动作枚举注释更新 (存量库; 新装库 modules_init 已是新注释)
-- ------------------------------------------------------------
ALTER TABLE `wf_task`
  MODIFY `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1=待办,2=已同意,3=已驳回,4=已转出,5=已作废,6=已失效(退回/撤销后原同意失效),7=已委派,8=委办完成';

ALTER TABLE `wf_instance`
  MODIFY `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1=运行中,2=已通过,4=已撤销,5=已终止,6=已退回(待重提),7=已撤回(发起人收回,待重提)';

ALTER TABLE `wf_record`
  MODIFY `action` VARCHAR(16) NOT NULL COMMENT '动作:submit/resubmit/approve/reject/back/cancel/withdraw/cc/finish/transfer/delegate/delegateResolve/terminate/urge/append/reduce/timeoutRemind/timeoutTransfer/timeoutApprove';

-- ------------------------------------------------------------
-- 3) API 资源 (撤回/委派/委派处理; 按钮权限复用既有 flow:instance:cancel / flow:task:handle)
-- ------------------------------------------------------------
INSERT IGNORE INTO `sys_api` (`path`,`method`,`group_name`,`description`) VALUES
  ('/api/v1/flow/instances/{id}/withdraw',    'POST', '审批中心', '撤回流程(发起人,尚无审批时收回待改)'),
  ('/api/v1/flow/tasks/{id}/delegate',        'POST', '审批中心', '委派(代办后回到原审批人终审)'),
  ('/api/v1/flow/tasks/{id}/delegateResolve', 'POST', '审批中心', '委派处理(被委托人提交意见)');

-- ------------------------------------------------------------
-- 4) 超时扫描定时任务 (幂等: 未种子过才插入; 启用态, 默认每 10 分钟)
-- ------------------------------------------------------------
INSERT INTO `sys_job` (`name`,`handler`,`cron_expr`,`params`,`status`,`remark`)
SELECT '流程超时扫描', 'flow.timeoutScan', '0 */10 * * * *', '', 1,
       '审批任务超时处理: 提醒/自动转办/自动通过 (按审批节点超时配置, 无配置不受影响)'
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `sys_job` WHERE `handler` = 'flow.timeoutScan' AND `deleted_at` IS NULL);

-- ============================================================
-- 清理 (演练完成后手动执行)
-- ============================================================
-- DELETE FROM `sys_job` WHERE `handler` = 'flow.timeoutScan' AND `deleted_at` IS NULL;
-- DELETE FROM `sys_api` WHERE `path` IN ('/api/v1/flow/instances/{id}/withdraw',
--   '/api/v1/flow/tasks/{id}/delegate', '/api/v1/flow/tasks/{id}/delegateResolve');
-- ALTER TABLE `wf_task` DROP INDEX `idx_status_due`, DROP COLUMN `delegate_from_id`, DROP COLUMN `due_time`;

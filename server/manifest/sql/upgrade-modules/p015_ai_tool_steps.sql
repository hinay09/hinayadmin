-- ============================================================
-- p015: AI 工具调用第一层增强 (存量库增量; 新装库执行 modules_init.sql 已含)
--
-- 内容:
--   1) ai_chat_message.role 注释扩为 user/assistant/tool —— 工具调用步骤以
--      role=tool 行落库 (内容为步骤 JSON: id/name/args/result/error),
--      随下一轮上下文回放 (模型跨轮记得调用过什么), 历史接口一并返回;
--   2) 新增 token 用量列 prompt_tokens / completion_tokens / total_tokens
--      (仅 assistant 行填写, 本轮累计; 网关未回报 usage 时为 0)。
--
-- 幂等性: ADD COLUMN 无 IF NOT EXISTS (MySQL 8 仍不支持), 重复执行会报
--   Duplicate column 而中止 —— 按序执行一次即可, 不可重放。
-- 依赖: modules_init.sql 的 ai_chat_message 建表 (原 p007)。
-- ============================================================

ALTER TABLE `ai_chat_message`
  MODIFY COLUMN `role` VARCHAR(16) NOT NULL COMMENT '角色:user/assistant/tool(tool=工具调用步骤,内容为步骤JSON)',
  ADD COLUMN `prompt_tokens`     INT UNSIGNED DEFAULT NULL COMMENT '输入token用量(仅assistant行,本轮累计)' AFTER `content`,
  ADD COLUMN `completion_tokens` INT UNSIGNED DEFAULT NULL COMMENT '输出token用量(仅assistant行,本轮累计)' AFTER `prompt_tokens`,
  ADD COLUMN `total_tokens`      INT UNSIGNED DEFAULT NULL COMMENT '总token用量(仅assistant行;网关未回报为0)'    AFTER `completion_tokens`;

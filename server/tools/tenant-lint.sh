#!/usr/bin/env bash
# tenant-lint.sh —— 多租户工程规约静态检查 (docs/multi-tenant.md §9/§11-P5, make tenant-lint)。
#
# 规约清单 (任一命中即 FAIL):
#   1. 业务代码对隔离表的手写 DML: tx.Exec / .Exec( / DoExec( / .Raw( 语句涉及 A 类表
#      (注入器 DoExec 已 fail-close, 此处提前在 CI 拦截写法);
#   2. A 类表使用 .Cache( (查询缓存在驱动层之前, 缓存 key 未注入租户 —— §3.2);
#   3. 业务代码手写 tenant_id 条件字面量 (租户条件只由注入器管理 —— §3.2);
#   4. SQL 基线新增外键约束 (无外键方针);
#   5. Redis 新增键未带租户维度的提示由评审把关 (现有键全为票据/会话类, 无法静态判定,
#      此处仅统计提示)。
set -uo pipefail
cd "$(dirname "$0")/.."

FAIL=0
hit() { FAIL=$((FAIL+1)); echo "✗ $1"; }

# A 类隔离表清单 (与 internal/logic/ormfill/tenantorm.go tenantTables 对齐)
TABLES='sys_user|sys_org|sys_role|sys_role_org|sys_post|sys_user_post|sys_config|sys_file|sys_audit_log|sys_login_log|sys_job|sys_job_log|wf_definition|wf_instance|wf_task|wf_record|biz_leave|biz_message|ai_conversation|ai_chat_message'

echo "== 1. 手写 DML 引用隔离表 (tx.Exec/.Exec(/DoExec(/.Raw() =="
# ormfill 包自身是实现体豁免; gencode 的 g.DB().Model("sys_menu") 为 B 类表不在清单
while IFS=: read -r file line text; do
  [ -z "${file:-}" ] && continue
  hit "$file:$line 手写 DML 涉及隔离表: $text"
done < <(grep -rnE '\.(Exec|Raw)\(|DoExec\(' \
  --include='*.go' internal/ api/ 2>/dev/null \
  | grep -v 'internal/logic/ormfill/' | grep -v '_test.go' \
  | grep -Ev '^\s*//' \
  | grep -E "$TABLES" | awk -F: '{print $1":"$2": "$3}')

echo "== 2. A 类表 .Cache( 使用 (含全部表, 全仓库零容忍) =="
while IFS=: read -r file line text; do
  [ -z "${file:-}" ] && continue
  hit "$file:$line 禁止 .Cache( (查询缓存绕过租户注入): $text"
done < <(grep -rnE '\.Cache\(' --include='*.go' internal/ api/ 2>/dev/null \
  | grep -v '_test.go' | awk -F: '{print $1":"$2": "$3}')

echo "== 3. 业务代码手写 tenant_id 条件字面量 =="
# 禁的是"租户条件" (tenant_id = 变量/字面量/入参); 平台行定位 Where("tenant_id",
# consts.TenantPlatformId) 是配置覆盖读序等场景的 sanctioned 例外, 放行。
while IFS=: read -r file line text; do
  [ -z "${file:-}" ] && continue
  hit "$file:$line 禁止手写 tenant_id 条件 (租户条件仅注入器管理): $text"
done < <(grep -rnE 'Where\(["'"'"'`][^"'"'"'`]*tenant_id' \
  --include='*.go' internal/logic internal/controller 2>/dev/null \
  | grep -v 'internal/logic/ormfill/' | grep -v '_test.go' \
  | grep -v 'internal/logic/tenant/' \
  | grep -v 'consts.TenantPlatformId' \
  | awk -F: '{print $1":"$2": "$3}')

echo "== 4. SQL 基线外键约束 (无外键方针) =="
while IFS=: read -r file line text; do
  [ -z "${file:-}" ] && continue
  hit "$file:$line 禁止新增外键约束: $text"
done < <(grep -rniE 'FOREIGN KEY|REFERENCES' manifest/sql/*.sql manifest/sql/demo/*.sql 2>/dev/null \
  | awk -F: '{print $1":"$2": "$3}')

echo "== 5. Redis 键清单 (人工复核提示) =="
grep -rhoE '"hinay:[a-z0-9:_-]+"' --include='*.go' internal/consts utility/ 2>/dev/null | sort -u | sed 's/^/  · /'

echo ""
if [ $FAIL -eq 0 ]; then
  echo "✔ tenant-lint 全部通过"
  exit 0
else
  echo "✗ tenant-lint 命中 $FAIL 处违规"
  exit 1
fi

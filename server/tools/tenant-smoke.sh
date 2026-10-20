#!/usr/bin/env bash
# tenant-smoke.sh —— 多租户后端快速冒烟 (开发自检; 完整验收矩阵见 docs/multi-tenant.md §12)
# 前置: 后端已启动 (make run), 本地库已执行 init.sql, multiTenant.enable=true,
#       sys.captcha_enable=false (或自行扩展脚本带验证码), 本机 python3 + pycryptodome。
# 幂等: 平台 admin 密码在 123456 / Admin@123456 之间自适应; 租户 acme 重复创建时报"已存在"视为通过。
set -uo pipefail
BASE=http://127.0.0.1:8000/api/v1
PASS=0; FAIL=0
ok()   { PASS=$((PASS+1)); echo "  ✓ $1"; }
bad()  { FAIL=$((FAIL+1)); echo "  ✗ $1"; }
# 用法1: jq_get "<json>" "<expr>"; 用法2(管道): cat json | jq_get "<expr>"
jq_get() {
  if [ $# -ge 2 ]; then
    python3 -c "import sys,json;r=json.load(sys.stdin);d=r.get('data',r) if isinstance(r,dict) else r;print(eval(sys.argv[1]))" "$2" 2>/dev/null <<<"$1"
  else
    python3 -c "import sys,json;r=json.load(sys.stdin);d=r.get('data',r) if isinstance(r,dict) else r;print(eval(sys.argv[1]))" "$1" 2>/dev/null
  fi
}

# rsa_login <username> <password> [tenantCode] — 输出登录响应 JSON
rsa_login() {
  local USER=$1 PASS=$2 TC=${3:-}
  local PUB KEYID PEM ENC BODY
  PUB=$(curl -s $BASE/auth/public-key)
  KEYID=$(jq_get "$PUB" "d['keyId']"); PEM=$(jq_get "$PUB" "d['publicKey']")
  ENC=$(python3 - "$PEM" "$PASS" <<'PYEOF'
import sys, base64
from Crypto.Cipher import PKCS1_v1_5 as C
from Crypto.PublicKey import RSA
k = RSA.import_key(sys.argv[1]); print(base64.b64encode(C.new(k).encrypt(sys.argv[2].encode())).decode())
PYEOF
)
  BODY="{\"username\":\"$USER\",\"password\":\"$ENC\",\"keyId\":\"$KEYID\""
  [ -n "$TC" ] && BODY="$BODY,\"tenantCode\":\"$TC\""
  BODY="$BODY}"
  curl -s -X POST $BASE/auth/login -H 'Content-Type: application/json' ${TC:+-H "X-Tenant-Code: $TC"} -d "$BODY"
}

echo "== 1. 平台超管登录 (首登自动完成强制改密, 幂等) =="
LOGIN=$(rsa_login admin 123456)
TOKEN=$(jq_get "$LOGIN" "d['token']")
if [ -n "$TOKEN" ]; then
  CP=$(curl -s -X PUT $BASE/auth/password -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
       -d '{"oldPassword":"123456","newPassword":"Admin@123456"}')
  ok "admin 首登改密: code=$(jq_get "$CP" "r['code']")"
  LOGIN=$(rsa_login admin 'Admin@123456')
  TOKEN=$(jq_get "$LOGIN" "d['token']")
else
  LOGIN=$(rsa_login admin 'Admin@123456')
  TOKEN=$(jq_get "$LOGIN" "d['token']")
fi
[ -n "$TOKEN" ] && ok "admin 登录成功 (tenantId=$(jq_get "$LOGIN" "d['userInfo']['tenantId']") isPlatformAdmin=$(jq_get "$LOGIN" "d['userInfo']['isPlatformAdmin']"))" || { bad "admin 登录失败: $LOGIN"; exit 1; }
AH="Authorization: Bearer $TOKEN"

echo "== 2. 创建租户 acme (六件套) =="
T=$(curl -s -X POST $BASE/system/tenants -H "$AH" -H 'Content-Type: application/json' \
  -d '{"name":"Acme租户","code":"acme","contact":"bob","adminUsername":"acmeadmin","menuIds":[30,40,50,51,52,15,10,11,60,9040,9350,9000]}')
TID=$(jq_get "$T" "d['id']")
if [ -n "$TID" ] && [ "$TID" != "None" ]; then
  ok "租户创建 id=$TID adminUid=$(jq_get "$T" "d['adminUserId']")"
else
  # 幂等重跑: 已存在则查回 id
  TID=$(curl -s "$BASE/system/tenants?keyword=acme" -H "$AH" | jq_get "d['list'][0]['id']")
  [ -n "$TID" ] && [ "$TID" != "None" ] && ok "租户 acme 已存在, 复用 id=$TID" || { bad "租户创建失败: $T"; exit 1; }
fi

echo "== 3. 租户管理员登录 (tenantCode=acme; 首登自动完成强制改密) =="
TL=$(rsa_login acmeadmin 123456 acme)
TTOKEN=$(jq_get "$TL" "d['token']")
if [ -n "$TTOKEN" ]; then
  MUST=$(jq_get "$TL" "d['userInfo']['mustChangePwd']")
  curl -s -X PUT $BASE/auth/password -H "Authorization: Bearer $TTOKEN" -H 'Content-Type: application/json' \
    -d '{"oldPassword":"123456","newPassword":"Acme@123456"}' > /dev/null
  TL=$(rsa_login acmeadmin 'Acme@123456' acme)
  TTOKEN=$(jq_get "$TL" "d['token']")
  ok "acmeadmin 首登 (mustChangePwd=$MUST) → 改密 → 重登 tenantId=$(jq_get "$TL" "d['userInfo']['tenantId']") isTenantAdmin=$(jq_get "$TL" "d['userInfo']['isTenantAdmin']")"
else
  TL=$(rsa_login acmeadmin 'Acme@123456' acme)
  TTOKEN=$(jq_get "$TL" "d['token']")
  [ -n "$TTOKEN" ] && ok "acmeadmin 登录成功 (已改密态) tenantId=$(jq_get "$TL" "d['userInfo']['tenantId']") isTenantAdmin=$(jq_get "$TL" "d['userInfo']['isTenantAdmin']")" || { bad "acmeadmin 登录失败: $TL"; exit 1; }
fi
TH="Authorization: Bearer $TTOKEN"

echo "== 4. 数据隔离 =="
curl -s -X POST $BASE/system/users -H "$AH" -H 'Content-Type: application/json' \
  -d '{"username":"platformuser1","password":"Pub@12345","nickname":"平台用户"}' > /dev/null
US=$(curl -s "$BASE/system/users?pageSize=100" -H "$TH")
HASP=$(jq_get "$US" "sum(1 for u in d['list'] if u['username']=='platformuser1')")
[ "$HASP" = "0" ] && ok "acme 视角看不到平台用户 (行级隔离生效)" || bad "acme 可见平台用户!"
ACME_ROWS=$(jq_get "$US" "len(d['list'])")
ok "acme 视角用户数=$ACME_ROWS (仅本租户)"
PUS=$(curl -s "$BASE/system/users?pageSize=100" -H "$AH")
PROWS=$(jq_get "$PUS" "len(d['list'])")
PTNAME=$(jq_get "$PUS" "next((u['tenantName'] for u in d['list'] if u['username']=='acmeadmin'), 'N/A')")
ok "平台视角用户数=$PROWS (全租户混排, acmeadmin 行租户名=$PTNAME)"

echo "== 4.5 其余隔离列表的租户归属 (§8.6 补齐) =="
# 岗位: acme 建一个岗位 (注入归属), 平台视角混排可见并带租户名
PC=$(curl -s -X POST $BASE/system/posts -H "$TH" -H 'Content-Type: application/json' \
  -d '{"postCode":"acme_hr","postName":"Acme人事","postKind":1,"sort":1,"status":1}')
PCC=$(jq_get "$PC" "r['code']")
[ "$PCC" = "0" ] && ok "acmeadmin 建岗位成功 (归属注入)" || bad "建岗位失败: $PC"
PL=$(curl -s "$BASE/system/posts?page=1&pageSize=100" -H "$AH")
PLT=$(jq_get "$PL" "next((p['tenantName'] for p in d['list'] if p['postCode']=='acme_hr'), 'N/A')")
[ "$PLT" = "Acme租户" ] && ok "岗位列表平台视角标注租户 (acme_hr → Acme租户)" || bad "岗位租户列异常: $PLT"
PLT2=$(curl -s "$BASE/system/posts?page=1&pageSize=100" -H "$TH" | jq_get "sum(1 for p in d['list'] if p['tenantName'] not in ('Acme租户',))")
[ "$PLT2" = "0" ] && ok "acme 视角岗位仅本租户 (注入隔离)" || bad "岗位列表泄漏"
# 组织树: 平台视角为多租户森林, 根节点带租户名
OT=$(curl -s "$BASE/system/orgs/tree" -H "$AH")
OTR=$(jq_get "$OT" "sum(1 for n in d['tree'] if n.get('tenantName')=='Acme租户')")
[ "$OTR" != "0" ] && ok "组织树根节点标注租户 (Acme租户森林根存在)" || bad "组织树租户标注缺失"
# 定时任务: 平台视角列表租户标注 (种子任务归属平台租户)
JL=$(curl -s "$BASE/system/jobs?page=1&pageSize=50" -H "$AH")
JLT=$(jq_get "$JL" "next((j['tenantName'] for j in d['list'] if j['handler']=='flow.timeoutScan'), 'N/A')")
[ "$JLT" = "平台租户" ] && ok "定时任务列表标注租户 (种子任务→平台租户)" || bad "任务租户列异常: $JLT"
# 岗位下拉收窄 (同 roles/all): 平台未切换仅平台岗位
PA=$(curl -s "$BASE/system/posts/all" -H "$AH")
PAM=$(jq_get "$PA" "sum(1 for p in d['list'] if p['tenantId'] not in (1,))")
[ "$PAM" = "0" ] && ok "平台超管未切换: 岗位下拉仅平台租户岗位 (不混排)" || bad "岗位下拉混入他租岗位 ($PAM 条)"

echo "== 5. 跨租户详情表现为不存在 =="
D=$(curl -s "$BASE/system/users/1" -H "$TH")
CODE=$(jq_get "$D" "r['code']")
[ "$CODE" != "0" ] && [ -n "$CODE" ] && ok "acme 查平台用户 id=1 → code=$CODE (不存在, 不泄露存在性)" || bad "跨租户详情意外可见: $D"

echo "== 6. 伪造 X-Tenant-Id 被忽略 =="
S=$(curl -s "$BASE/system/users?pageSize=5" -H "$TH" -H "X-Tenant-Id: 1")
SC=$(jq_get "$S" "r['code']")
SHAS=$(jq_get "$S" "sum(1 for u in d['list'] if u['username']=='platformuser1')")
[ "$SC" = "0" ] && [ "$SHAS" = "0" ] && ok "伪造头被忽略: 请求成功且数据面仍=acme (已写告警审计)" || bad "伪造头处理异常: $S"

echo "== 7. 平台超管代运维切换 =="
SW=$(curl -s "$BASE/system/users?pageSize=100" -H "$AH" -H "X-Tenant-Id: $TID")
SWROWS=$(jq_get "$SW" "len(d['list'])")
SWHAS=$(jq_get "$SW" "sum(1 for u in d['list'] if u['username']=='platformuser1')")
[ "$SWHAS" = "0" ] && [ -n "$SWROWS" ] && ok "切换到 acme 后用户数=$SWROWS, 平台用户不可见 (数据面=目标租户)" || bad "切换后数据面异常: $SW"
MENUS=$(curl -s "$BASE/auth/menus" -H "$AH" -H "X-Tenant-Id: $TID")
HAST=$(jq_get "$MENUS" "sum(1 for m in d['menus'] if m['path']=='/system/tenants')")
HASMON=$(jq_get "$MENUS" "sum(1 for m in d['menus'] if m['path']=='/system/monitor')")
[ "$HAST" = "0" ] && [ "$HASMON" = "0" ] && ok "切换后平台管理入口 (租户管理/监控) 自动隐藏" || bad "切换后平台菜单泄漏!"
# 切换态身份面回归: 会话端点操作本人行 (归属租户), 不受数据面注入影响 (§18.3 修正5)
UI=$(curl -s "$BASE/auth/userInfo" -H "$AH" -H "X-Tenant-Id: $TID")
UIC=$(jq_get "$UI" "r['code']")
SWNAME=$(jq_get "$UI" "d['switchTenant']['name']")
UIROLES=$(jq_get "$UI" "len(d['roles'])")
[ "$UIC" = "0" ] && ok "切换态 userInfo 正常 (身份面): switchTenant.name=$SWNAME" || bad "切换态 userInfo 失败: $UI"
[ -n "$SWNAME" ] && [ "$SWNAME" != "None" ] && ok "切换态徽标数据齐全 (switchTenant.name 非空)" || bad "switchTenant.name 为空"
[ "$UIROLES" != "0" ] && ok "切换态角色仍展示归属域真实角色 ($UIROLES 个)" || bad "切换态角色丢失"
PR=$(curl -s -X PUT $BASE/auth/profile -H "$AH" -H "X-Tenant-Id: $TID" -H 'Content-Type: application/json' \
  -d "{\"nickname\":\"$(curl -s "$BASE/auth/userInfo" -H "$AH" | jq_get "d['nickname']")\",\"avatar\":\"\",\"email\":\"\",\"phone\":\"\"}")
ok "切换态改资料 code=$(jq_get "$PR" "r['code']") (身份面写本人行)"
# 切换态 refresh (旧 token 黑名单机制会轮换 token, 用新 token 续验)
RT=$(curl -s -X POST $BASE/auth/refresh -H "$AH" -H "X-Tenant-Id: $TID")
RTC=$(jq_get "$RT" "r['code']")
[ "$RTC" = "0" ] && ok "切换态 token 续签正常 (身份面)" || bad "切换态 refresh 失败: $RT"
NEWTOK=$(jq_get "$RT" "d['token']")
if [ -n "$NEWTOK" ] && [ "$NEWTOK" != "None" ]; then TOKEN="$NEWTOK"; AH="Authorization: Bearer $TOKEN"; fi

echo "== 8. 应用授权集双向联动 =="
SM=$(curl -s -X PUT "$BASE/system/tenants/$TID/menus" -H "$AH" -H 'Content-Type: application/json' \
  -d '{"menuIds":[30,40,50,51,52,15,10,11,60,9040,9350]}')
ok "收缩授权集 (移除审批中心 9000): code=$(jq_get "$SM" "r['code']")"
MENUS2=$(curl -s "$BASE/auth/menus" -H "$TH")
HASFLOW=$(jq_get "$MENUS2" "sum(1 for m in d['menus'] if m['path']=='/flow')")
[ "$HASFLOW" = "0" ] && ok "acmeadmin 菜单中审批中心已消失 (收缩即时生效)" || bad "审批中心菜单仍在!"
RID=$(curl -s "$BASE/system/roles/all" -H "$TH" | jq_get "[r['id'] for r in d['list'] if r['code']=='tenant_admin'][0]")
GM=$(curl -s "$BASE/system/roles/$RID/menus" -H "$TH")
HAS9010=$(jq_get "$GM" "sum(1 for m in d['menuIds'] if m==9010)")
[ "$HAS9010" = "0" ] && ok "tenant_admin 角色策略被联动清理 (9010 流程定义页移除)" || bad "9010 策略残留!"
curl -s -X PUT "$BASE/system/tenants/$TID/menus" -H "$AH" -H 'Content-Type: application/json' \
  -d '{"menuIds":[30,40,50,51,52,15,10,11,60,9040,9350,9000]}' > /dev/null
MENUS3=$(curl -s "$BASE/auth/menus" -H "$TH")
HASFLOW2=$(jq_get "$MENUS3" "sum(1 for m in d['menus'] if m['path']=='/flow')")
[ "$HASFLOW2" != "0" ] && ok "扩张后 tenant_admin 立即可见审批中心 (买了就用)" || bad "扩张联动失败!"

echo "== 8.5 角色归属展示与下拉收口 =="
RL=$(curl -s "$BASE/system/roles?pageSize=100" -H "$AH")
RLT=$(jq_get "$RL" "next((r['tenantName'] for r in d['list'] if r['code']=='tenant_admin' and r['tenantName']=='Acme租户'), 'N/A')")
[ "$RLT" = "Acme租户" ] && ok "平台视角角色列表含租户列 (acme 的 tenant_admin 标注 Acme租户)" || bad "角色列表租户列缺失: $RLT"
RALL_P=$(curl -s "$BASE/system/roles/all" -H "$AH")
MIXP=$(jq_get "$RALL_P" "sum(1 for r in d['list'] if r['tenantId'] not in (1,))")
[ "$MIXP" = "0" ] && ok "平台超管未切换: 角色下拉仅平台租户角色 (不混排)" || bad "角色下拉混入其他租户角色 ($MIXP 条)"
RALL_T=$(curl -s "$BASE/system/roles/all" -H "$TH")
MIXT=$(jq_get "$RALL_T" "sum(1 for r in d['list'] if r['tenantId'] not in ($TID,))")
[ "$MIXT" = "0" ] && ok "acmeadmin: 角色下拉仅本租户角色" || bad "租户角色下拉泄漏 ($MIXT 条)"
RALL_S=$(curl -s "$BASE/system/roles/all" -H "$AH" -H "X-Tenant-Id: $TID")
MIXS=$(jq_get "$RALL_S" "sum(1 for r in d['list'] if r['tenantId'] != $TID)")
[ "$MIXS" = "0" ] && ok "平台超管切换态: 角色下拉=目标租户角色" || bad "切换态下拉混排 ($MIXS 条)"

echo "== 8.6 租户套餐: 模板复用/绑定保护/同步联动 (§7.2) =="
PKG=$(curl -s -X POST $BASE/system/tenant-packages -H "$AH" -H 'Content-Type: application/json' \
  -d '{"name":"冒烟协作包","menuIds":[30,40,50,51,52,15,10,11,60,9040,9350],"status":1,"remark":"smoke"}')
PKGID=$(jq_get "$PKG" "d['id']")
if [ -n "$PKGID" ] && [ "$PKGID" != "None" ]; then
  ok "套餐创建 id=$PKGID (授权集快照 11 项, 不含审批中心)"
else
  PKGID=$(curl -s "$BASE/system/tenant-packages?page=1&pageSize=100" -H "$AH" | jq_get "next((p['id'] for p in d['list'] if p['name']=='冒烟协作包'), None)")
  [ -n "$PKGID" ] && [ "$PKGID" != "None" ] && ok "套餐已存在, 复用 id=$PKGID" || { bad "套餐创建失败: $PKG"; }
fi
BADPKG=$(curl -s -X POST $BASE/system/tenant-packages -H "$AH" -H 'Content-Type: application/json' \
  -d '{"name":"冒烟越权包","menuIds":[9300]}')
BADPKGC=$(jq_get "$BADPKG" "r['code']")
[ "$BADPKGC" != "0" ] && ok "套餐含平台专属菜单(9300)被拒 → code=$BADPKGC" || bad "平台专属菜单进了套餐模板!"
# 绑定 (仅记录归属, 不自动改授权集)
BND=$(curl -s -X PUT "$BASE/system/tenants/$TID" -H "$AH" -H 'Content-Type: application/json' \
  -d '{"name":"Acme租户","contact":"bob","status":1,"remark":"","plan":"standard","packageId":'"$PKGID"'}')
BNDC=$(jq_get "$BND" "r['code']")
TLIST=$(curl -s "$BASE/system/tenants?keyword=acme" -H "$AH")
TPKG=$(jq_get "$TLIST" "next((t['packageName'] for t in d['list'] if t['id']==$TID), '')")
[ "$BNDC" = "0" ] && [ "$TPKG" = "冒烟协作包" ] && ok "套餐绑定成功且列表标注归属 (packageName=$TPKG)" || bad "套餐绑定异常: code=$BNDC pkg=$TPKG"
# 使用中禁删 (50031)
DELPKG=$(curl -s -X DELETE "$BASE/system/tenant-packages/$PKGID" -H "$AH")
DELPC=$(jq_get "$DELPKG" "r['code']")
[ "$DELPC" = "50031" ] && ok "使用中套餐删除被拒 → 50031" || bad "使用中套餐被删: $DELPKG"
# 同步: 套餐不含审批中心(9000), 同步后 acme 收缩 (与 SetMenus 同路径联动)
SYNC=$(curl -s -X POST "$BASE/system/tenant-packages/$PKGID/sync" -H "$AH")
SYNCC=$(jq_get "$SYNC" "r['code']"); SYNCN=$(jq_get "$SYNC" "d['count']")
MENUSP=$(curl -s "$BASE/auth/menus" -H "$TH")
HASFLOWP=$(jq_get "$MENUSP" "sum(1 for m in d['menus'] if m['path']=='/flow')")
[ "$SYNCC" = "0" ] && [ "$SYNCN" = "1" ] && [ "$HASFLOWP" = "0" ] && ok "套餐同步 count=1, 审批中心被收缩" || bad "套餐同步异常: $SYNC / flow=$HASFLOWP"
# 解绑后可删
curl -s -X PUT "$BASE/system/tenants/$TID" -H "$AH" -H 'Content-Type: application/json' \
  -d '{"name":"Acme租户","contact":"bob","status":1,"remark":"","plan":"standard","packageId":0}' > /dev/null
DELPKG2=$(curl -s -X DELETE "$BASE/system/tenant-packages/$PKGID" -H "$AH")
DELPC2=$(jq_get "$DELPKG2" "r['code']")
[ "$DELPC2" = "0" ] && ok "解绑后套餐删除成功" || bad "解绑后删除失败: $DELPKG2"

echo "== 9. 租户管理员可绑清单收口 =="
TREES=$(curl -s "$BASE/system/menus/tree" -H "$TH")
HASM=$(jq_get "$TREES" "sum(1 for m in d['tree'] if m['id']==22)")
[ "$HASM" = "0" ] && ok "acme 可绑菜单树不含服务器监控 (22)" || bad "可绑树泄漏平台专属菜单!"
APIS=$(curl -s "$BASE/system/apis/all" -H "$TH")
HASMONAPI=$(jq_get "$APIS" "sum(1 for a in d['list'] if a['path']=='/api/v1/system/monitor/server')")
[ "$HASMONAPI" = "0" ] && ok "acme 可绑 API 清单不含平台监控接口" || bad "可绑 API 泄漏平台接口!"
BM=$(curl -s -X PUT "$BASE/system/roles/$RID/menus" -H "$TH" -H 'Content-Type: application/json' -d '{"menuIds":[22]}')
BMC=$(jq_get "$BM" "r['code']")
[ "$BMC" != "0" ] && ok "绑平台专属菜单被拒 → code=$BMC" || bad "平台菜单绑定未被拒!"
BA=$(curl -s -X PUT "$BASE/system/roles/$RID/apis" -H "$TH" -H 'Content-Type: application/json' \
  -d '{"apis":[{"path":"/api/v1/system/tenants","method":"GET"}]}')
BAC=$(jq_get "$BA" "r['code']")
[ "$BAC" != "0" ] && ok "绑租户管理 API 被拒 → code=$BAC" || bad "平台 API 绑定未被拒!"

echo "== 10. 配置覆盖读序 =="
curl -s -X POST $BASE/system/configs -H "$TH" -H 'Content-Type: application/json' \
  -d '{"configKey":"sys.name","configValue":"Acme 专属站点","configType":0,"name":"系统名称","status":1}' > /dev/null
CA=$(curl -s "$BASE/system/configs/all" -H "$TH")
CNAME=$(jq_get "$CA" "d['list']['sys.name']['value']")
[ "$CNAME" = "Acme 专属站点" ] && ok "acme 读到租户覆盖站点名" || bad "覆盖读序失败: $CNAME"
CP=$(curl -s "$BASE/system/configs/all" -H "$AH")
CPNAME=$(jq_get "$CP" "d['list']['sys.name']['value']")
[ "$CPNAME" = "Hinay Admin" ] && ok "平台侧仍是平台默认值" || bad "平台侧被污染: $CPNAME"
CTRY=$(curl -s -X POST $BASE/system/configs -H "$TH" -H 'Content-Type: application/json' \
  -d '{"configKey":"ai.api_key","configValue":"hack","configType":0,"name":"x","status":1}')
CTRYC=$(jq_get "$CTRY" "r['code']")
[ "$CTRYC" != "0" ] && ok "租户建非白名单键被拒 → code=$CTRYC" || bad "非白名单键未被拒!"
CMT=$(jq_get "$CA" "d['list'].get('sys.multi_tenant',{}).get('value')")
ok "configs/all 下发 multi_tenant=$CMT (前端感知)"
RC=$(curl -s -X POST "$BASE/system/tenants/$TID/configs/reset" -H "$AH")
CA2=$(curl -s "$BASE/system/configs/all" -H "$TH")
CNAME2=$(jq_get "$CA2" "d['list']['sys.name']['value']")
[ "$CNAME2" = "Hinay Admin" ] && ok "平台重置覆盖配置后回归默认" || bad "重置失败: $CNAME2"

echo "== 11. 公开租户信息 + 登录三态错误 =="
TI=$(curl -s "$BASE/auth/tenant/info?code=acme")
ok "tenant/info acme → found=$(jq_get "$TI" "d['found']") name=$(jq_get "$TI" "d['name']") multiTenant=$(jq_get "$TI" "d['multiTenant']")"
TIN=$(curl -s "$BASE/auth/tenant/info?code=nosuch")
TINF=$(jq_get "$TIN" "d['found']")
[ "$TINF" = "False" ] && ok "不存在的租户码 → found=False (不泄露信息)" || bad "枚举面异常: $TIN"
BADCODE=$(curl -s -X POST $BASE/auth/login -H 'Content-Type: application/json' -d '{"username":"x","password":"y","keyId":"z","tenantCode":"nosuch"}')
BCC=$(jq_get "$BADCODE" "r['code']")
[ "$BCC" = "50023" ] && ok "无效租户码登录 → 50023" || bad "无效租户码错误码: $BCC"

echo "== 12. 禁用租户: 登录与已登录会话全拦截 =="
curl -s -X PUT "$BASE/system/tenants/$TID" -H "$AH" -H 'Content-Type: application/json' \
  -d '{"name":"Acme租户","contact":"bob","status":0,"remark":"","plan":"standard"}' > /dev/null
BL=$(rsa_login acmeadmin 'Acme@123456' acme)
BLC=$(jq_get "$BL" "r['code']")
[ "$BLC" = "50024" ] && ok "禁用后登录拦截 → 50024" || bad "禁用后登录错误码: $BLC ($BL)"
BLK=$(curl -s "$BASE/system/users?pageSize=5" -H "$TH")
BLKC=$(jq_get "$BLK" "r['code']")
[ "$BLKC" = "50024" ] && ok "禁用后已登录会话拦截 → 50024" || bad "已登录拦截错误码: $BLKC"
curl -s -X PUT "$BASE/system/tenants/$TID" -H "$AH" -H 'Content-Type: application/json' \
  -d '{"name":"Acme租户","contact":"bob","status":1,"remark":"","plan":"standard"}' > /dev/null
OKAGAIN=$(curl -s "$BASE/system/users?pageSize=5" -H "$TH")
OKC=$(jq_get "$OKAGAIN" "r['code']")
[ "$OKC" = "0" ] && ok "恢复启用后会话恢复" || bad "恢复失败: $OKAGAIN"

echo "== 13. 到期租户拦截 =="
curl -s -X PUT "$BASE/system/tenants/$TID" -H "$AH" -H 'Content-Type: application/json' \
  -d '{"name":"Acme租户","contact":"bob","status":1,"remark":"","plan":"standard","expireAt":"2020-01-01"}' > /dev/null
EXPK=$(curl -s "$BASE/system/users?pageSize=5" -H "$TH")
EXPKC=$(jq_get "$EXPK" "r['code']")
[ "$EXPKC" = "50025" ] && ok "到期租户会话拦截 → 50025" || bad "到期拦截错误码: $EXPKC"
curl -s -X PUT "$BASE/system/tenants/$TID" -H "$AH" -H 'Content-Type: application/json' \
  -d '{"name":"Acme租户","contact":"bob","status":1,"remark":"","plan":"standard","expireAt":""}' > /dev/null
OKFIN=$(curl -s "$BASE/system/users?pageSize=5" -H "$TH")
OKFC=$(jq_get "$OKFIN" "r['code']")
[ "$OKFC" = "0" ] && ok "恢复永久后会话恢复" || bad "恢复失败: $OKFIN"

echo "== 14. 删除租户 (软删+code改写防撞uk) =="
T2=$(curl -s -X POST $BASE/system/tenants -H "$AH" -H 'Content-Type: application/json' \
  -d '{"name":"Temp租户","code":"temp","adminUsername":"tempadmin"}')
T2ID=$(jq_get "$T2" "d['id']")
DEL=$(curl -s -X DELETE "$BASE/system/tenants/$T2ID" -H "$AH")
DELC=$(jq_get "$DEL" "r['code']")
[ "$DELC" = "0" ] && ok "删除 temp → 成功" || bad "删除失败: $DEL"
RECREATE=$(curl -s -X POST $BASE/system/tenants -H "$AH" -H 'Content-Type: application/json' \
  -d '{"name":"Temp2","code":"temp","adminUsername":"tempadmin2"}')
RCC=$(jq_get "$RECREATE" "r['code']")
[ "$RCC" = "0" ] && ok "同 code 重建成功 (软删改写不撞uk)" || bad "同码重建失败: $RECREATE"
DEL2ID=$(jq_get "$RECREATE" "d['id']")
curl -s -X DELETE "$BASE/system/tenants/$DEL2ID" -H "$AH" > /dev/null
DEL1=$(curl -s -X DELETE "$BASE/system/tenants/1" -H "$AH")
DEL1C=$(jq_get "$DEL1" "r['code']")
[ "$DEL1C" != "0" ] && ok "平台租户(id=1)不可删除 → code=$DEL1C" || bad "平台租户被删除!"

echo "== 15. 租户管理员写本租户数据 (注入归属) =="
CU=$(curl -s -X POST $BASE/system/users -H "$TH" -H 'Content-Type: application/json' \
  -d '{"username":"acmeuser1","password":"AcmeU@12345","nickname":"Acme员工","roleIds":[]}')
CUROW=$(jq_get "$CU" "d['id']")
[ -n "$CUROW" ] && [ "$CUROW" != "None" ] && ok "acmeadmin 建用户成功 id=$CUROW (归属 acme)" || bad "建用户失败: $CU"
CUCHK=$(curl -s "$BASE/system/users?keyword=acmeuser1" -H "$AH" | jq_get "[u for u in d['list'] if u['username']=='acmeuser1'][0]['tenantName']")
[ "$CUCHK" = "Acme租户" ] && ok "平台视角确认 acmeuser1 归属 Acme租户" || bad "归属异常: $CUCHK"
SAME=$(curl -s -X POST $BASE/system/users -H "$AH" -H 'Content-Type: application/json' \
  -d '{"username":"acmeuser1","password":"Pub@12345","nickname":"同名"}')
SAMEC=$(jq_get "$SAME" "r['code']")
[ "$SAMEC" = "0" ] && ok "平台租户建同名用户 acmeuser1 → code=$SAMEC (租户内唯一, 跨租户不冲突)" || bad "跨租户同名冲突: $SAME"

echo "== 16. 租户管理员碰不到平台功能 =="
TM=$(curl -s -X POST $BASE/system/tenants -H "$TH" -H 'Content-Type: application/json' \
  -d '{"name":"x","code":"xx","adminUsername":"xxadmin"}')
TMC=$(jq_get "$TM" "r['code']")
[ "$TMC" != "0" ] && ok "租户管理员创建租户被拒 → code=$TMC" || bad "租户管理员可创建租户!"
MN=$(curl -s -X POST $BASE/system/menus -H "$TH" -H 'Content-Type: application/json' \
  -d '{"name":"x","type":2,"path":"/x","sort":1,"status":1}')
MNC=$(jq_get "$MN" "r['code']")
[ "$MNC" != "0" ] && ok "租户管理员维护菜单目录被拒 → code=$MNC" || bad "菜单目录可被租户维护!"
MON=$(curl -s "$BASE/system/monitor/server" -H "$TH")
MONC=$(jq_get "$MON" "r['code']")
[ "$MONC" != "0" ] && ok "租户管理员访问服务器监控被拒 → code=$MONC" || bad "监控接口泄漏!"
TPL=$(curl -s "$BASE/system/tenant-packages?page=1&pageSize=10" -H "$TH")
TPLC=$(jq_get "$TPL" "r['code']")
[ "$TPLC" != "0" ] && ok "租户管理员访问租户套餐被拒 → code=$TPLC" || bad "租户套餐接口泄漏给租户!"

echo ""
echo "冒烟结果: PASS=$PASS FAIL=$FAIL"
[ $FAIL -eq 0 ] && exit 0 || exit 1

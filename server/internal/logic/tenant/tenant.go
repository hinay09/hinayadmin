// Package tenant 租户生命周期与应用授权集 (平台超管专属能力)。
//
// 设计要点 (docs/multi-tenant.md §7/§7.1):
//   - 创建 = 六件套单事务: 租户行/组织根/租户管理员角色(按授权集生成 p 行)/
//     管理员账号(强制改密)/授权集物化(sys_tenant_menu)/管理员 g 行;
//     明确不含: 默认岗位/流程定义克隆/配置覆盖行;
//   - 授权集变更 = 重写 sys_tenant_menu + casbin 双向联动 同一事务, Reload 在提交后
//     (收缩删域内越界 p 行即时生效; 扩张只给 tenant_admin 角色追加 —— "买了就用");
//   - 内置保护: 平台租户(1)不可禁用/删除; tenant_admin 角色体系不可删/禁/code 变更
//     (角色表的保护在 system/role.go, 此处管租户行);
//   - 软删时事务内将 code 改写为 {code}#del{unix}, 防同码重建撞 uk;
//   - 所有入口双重守卫: multiTenant.enable 开启 + 平台超管身份, 并以平台视角
//     (WithoutTenant) 读写 —— 平台超管即使处于代运维切换态也不得误写目标租户授权。
package tenant

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/logic/casbinx"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/password"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/tenantx"
	"hinay.cn/admin/utility/xerror"
)

type sTenant struct{}

func init() {
	service.RegisterTenant(NewTenant())
}

func NewTenant() *sTenant {
	return &sTenant{}
}

// guard 平台守卫: 多租户开启 + 平台超管; 返回平台视角 ctx (屏蔽代运维切换注入)。
func guard(ctx context.Context) (context.Context, error) {
	if !tenantx.Enabled(ctx) {
		return nil, xerror.New(xerror.CodeMultiTenantDisabled)
	}
	if !contextx.IsPlatformAdmin(ctx) {
		return nil, xerror.New(xerror.CodeForbidden, "仅平台超管可操作租户管理")
	}
	return contextx.WithoutTenant(ctx), nil
}

// defaultTemplateMenuIds 默认授权模板 (基础协作包, 与 init.sql sys_tenant_menu 平台行种子一致)。
var defaultTemplateMenuIds = []uint64{30, 40, 50, 51, 52, 15, 10, 11, 60, 9040, 9350}

// TenantItem 列表行。
type TenantItem struct {
	Id           uint64      `json:"id"`
	Name         string      `json:"name"`
	Code         string      `json:"code"`
	Contact      string      `json:"contact"`
	Plan         string      `json:"plan"`
	QuotaUsers   int         `json:"quotaUsers"`
	QuotaStorage int64       `json:"quotaStorage"`
	ExpireAt     *gtime.Time `json:"expireAt"`
	Status       int         `json:"status"`
	Remark       string      `json:"remark"`
	CreatedAt    *gtime.Time `json:"createdAt"`
}

// List 租户分页列表 (平台视角全租户混排)。
func (s *sTenant) List(ctx context.Context, req *v1.TenantListReq) (res *v1.TenantListRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	q := dao.SysTenant.Ctx(ctx).Where("deleted_at IS NULL")
	if req.Keyword != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		q = q.Where("(name LIKE ? OR code LIKE ? OR contact LIKE ?)", kw, kw, kw)
	}
	if req.Status != nil {
		q = q.Where("status", *req.Status)
	}
	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "查询租户失败")
	}
	var rows []*model.SysTenant
	if err = q.Page(req.Page, req.PageSize).Order("id ASC").Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "查询租户失败")
	}
	items := make([]*TenantItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, &TenantItem{
			Id: r.Id, Name: r.Name, Code: r.Code, Contact: r.Contact, Plan: r.Plan,
			QuotaUsers: r.QuotaUsers, QuotaStorage: r.QuotaStorage, ExpireAt: r.ExpireAt,
			Status: r.Status, Remark: r.Remark, CreatedAt: r.CreatedAt,
		})
	}
	page := response.Page(items, int64(total), req.Page, req.PageSize)
	out := v1.TenantListRes(page)
	return &out, nil
}

// ListAll 启用租户全量 (平台切换器下拉)。
func (s *sTenant) ListAll(ctx context.Context, req *v1.TenantListAllReq) (res *v1.TenantListAllRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.activeTenantRows(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.TenantListAllRes{List: rows}, nil
}

// Detail 租户详情。
func (s *sTenant) Detail(ctx context.Context, req *v1.TenantDetailReq) (res *v1.TenantDetailRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	row, err := s.getTenant(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.TenantDetailRes{Tenant: row}, nil
}

// getTenant 按 ID 取租户行 (未删), 不存在报业务错误。
func (s *sTenant) getTenant(ctx context.Context, id uint64) (*model.SysTenant, error) {
	var row *model.SysTenant
	if err := dao.SysTenant.Ctx(ctx).Where("id", id).Where("deleted_at IS NULL").Scan(&row); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "查询租户失败")
	}
	if row == nil {
		return nil, xerror.New(xerror.CodeNotFound, "租户不存在")
	}
	return row, nil
}

// Create 创建租户 (向导六件套单事务)。
func (s *sTenant) Create(ctx context.Context, req *v1.TenantCreateReq) (res *v1.TenantCreateRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	// code 全局唯一预检 (软删行已改写 code, 活行查重即可)
	if n, _ := dao.SysTenant.Ctx(ctx).Where("code", req.Code).Where("deleted_at IS NULL").Count(); n > 0 {
		return nil, xerror.New(xerror.CodeTenantCodeExists)
	}
	menuIds := req.MenuIds
	if len(menuIds) == 0 {
		menuIds = defaultTemplateMenuIds
	}
	menus, err := s.loadMenus(ctx)
	if err != nil {
		return nil, err
	}
	menuSet := dedupU64(menuIds)
	if verr := s.validateGrantSet(menus, menuSet); verr != nil {
		return nil, verr
	}
	closure := closureOf(menus, menuSet)
	apiRows, err := s.allowedApis(ctx, closure)
	if err != nil {
		return nil, err
	}
	// 管理员账号在租户内唯一 (新建租户必然为空, 仅防极端并发撞 uk)
	pwdPlain := req.AdminPassword
	if pwdPlain == "" {
		pwdPlain = "123456"
	}
	hash, herr := password.Hash(pwdPlain)
	if herr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, herr, "密码加密失败")
	}
	nickname := req.AdminNickname
	if nickname == "" {
		nickname = req.AdminUsername
	}
	expire := parseExpire(req.ExpireAt)

	var tid, adminUid, roleRid uint64
	txErr := dao.SysTenant.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// ① 租户行
		tidv, err := dao.SysTenant.Ctx(ctx).Data(g.Map{
			"name": req.Name, "code": req.Code, "contact": req.Contact, "plan": req.Plan,
			"quota_users": 0, "quota_storage": 0, "expire_at": expire,
			"status": consts.StatusEnabled, "remark": req.Remark,
		}).InsertAndGetId()
		if err != nil {
			return err
		}
		tid = uint64(tidv)
		// ② 组织根
		orgId, err := dao.SysOrg.Ctx(ctx).Data(g.Map{
			"tenant_id": tid, "parent_id": 0, "name": req.Name, "sort": 1, "status": consts.StatusEnabled,
		}).InsertAndGetId()
		if err != nil {
			return err
		}
		_ = orgId
		// ③ 租户管理员角色 (域内 data_scope=全部, "圈养版超管")
		ridv, err := dao.SysRole.Ctx(ctx).Data(g.Map{
			"tenant_id": tid, "name": "租户管理员", "code": consts.RoleTenantAdminCode,
			"sort": 1, "status": consts.StatusEnabled, "remark": "租户管理员 (创建租户时生成)", "data_scope": consts.DataScopeAll,
		}).InsertAndGetId()
		if err != nil {
			return err
		}
		roleRid = uint64(ridv)
		dom := strconv.FormatUint(tid, 10)
		if err = insertRolePolicies(ctx, roleRid, dom, closure, apiRows); err != nil {
			return err
		}
		// ④ 管理员账号 (强制改密, 复用 must_change_pwd 机制)
		uidv, err := dao.SysUser.Ctx(ctx).Data(g.Map{
			"tenant_id": tid, "username": req.AdminUsername, "password": hash, "nickname": nickname,
			"org_id": uint64(orgId), "status": consts.StatusEnabled, "must_change_pwd": 1,
		}).InsertAndGetId()
		if err != nil {
			return err
		}
		adminUid = uint64(uidv)
		// ⑤ 授权集物化
		if err = rewriteTenantMenus(ctx, tid, menuSet); err != nil {
			return err
		}
		// ⑥ 管理员挂角色 (g 行, 域=新租户)
		if _, err = dao.CasbinRule.Ctx(ctx).Data(g.Map{
			"ptype": "g", "v0": strconv.FormatUint(adminUid, 10),
			"v1": strconv.FormatUint(roleRid, 10), "v2": dom,
		}).Insert(); err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, txErr, "创建租户失败")
	}
	if _, rerr := casbinx.Reload(ctx); rerr != nil {
		g.Log().Warningf(ctx, "casbin reload after tenant create failed: %v", rerr)
	}
	return &v1.TenantCreateRes{Id: tid, AdminUserId: adminUid}, nil
}

// Update 修改租户 (code 不可变; 平台租户不可禁用/设到期)。
func (s *sTenant) Update(ctx context.Context, req *v1.TenantUpdateReq) (res *v1.TenantUpdateRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = s.getTenant(ctx, req.Id); err != nil {
		return nil, err
	}
	if req.Id == consts.TenantPlatformId && (req.Status != consts.StatusEnabled || req.ExpireAt != "") {
		return nil, xerror.New(xerror.CodeBusinessError, "平台租户不可禁用或设置到期")
	}
	data := g.Map{
		"name": req.Name, "contact": req.Contact, "plan": req.Plan,
		"status": req.Status, "remark": req.Remark,
	}
	if req.ExpireAt == "" {
		data["expire_at"] = nil
	} else {
		if t := parseExpire(req.ExpireAt); t != nil {
			data["expire_at"] = t
		}
	}
	if _, err = dao.SysTenant.Ctx(ctx).Where("id", req.Id).Data(data).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "修改租户失败")
	}
	return &v1.TenantUpdateRes{}, nil
}

// Delete 删除租户 (软删 + code 原子改写防同码重建撞 uk; 平台租户不可删)。
func (s *sTenant) Delete(ctx context.Context, req *v1.TenantDeleteReq) (res *v1.TenantDeleteRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	if req.Id == consts.TenantPlatformId {
		return nil, xerror.New(xerror.CodeBusinessError, "平台租户不可删除")
	}
	row, err := s.getTenant(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	newCode := fmt.Sprintf("%s#del%d", row.Code, time.Now().Unix())
	txErr := dao.SysTenant.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := dao.SysTenant.Ctx(ctx).Where("id", req.Id).Data(g.Map{
			"deleted_at": gtime.Now(), "code": newCode,
		}).Update(); err != nil {
			return err
		}
		// 授权集与域内策略一并清理 (租户已不可登录, 遗留死策略污染平台视角)
		if _, err := dao.SysTenantMenu.Ctx(ctx).Where("tenant_id", req.Id).Delete(); err != nil {
			return err
		}
		if _, err := dao.CasbinRule.Ctx(ctx).Where("v3", strconv.FormatUint(req.Id, 10)).Delete(); err != nil {
			return err
		}
		if _, err := dao.CasbinRule.Ctx(ctx).
			Where("ptype", "g").Where("v2", strconv.FormatUint(req.Id, 10)).Delete(); err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, txErr, "删除租户失败")
	}
	if _, rerr := casbinx.Reload(ctx); rerr != nil {
		g.Log().Warningf(ctx, "casbin reload after tenant delete failed: %v", rerr)
	}
	return &v1.TenantDeleteRes{}, nil
}

// ResetConfigs 一键重置租户覆盖配置 (删租户行即回归平台默认)。
func (s *sTenant) ResetConfigs(ctx context.Context, req *v1.TenantResetConfigsReq) (res *v1.TenantResetConfigsRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = s.getTenant(ctx, req.Id); err != nil {
		return nil, err
	}
	if _, err = dao.SysConfig.Ctx(ctx).Where("tenant_id", req.Id).Delete(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "重置租户配置失败")
	}
	return &v1.TenantResetConfigsRes{}, nil
}

// GetMenus 获取租户应用授权集 (目录/页面级原值)。
func (s *sTenant) GetMenus(ctx context.Context, req *v1.TenantGetMenusReq) (res *v1.TenantGetMenusRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = s.getTenant(ctx, req.Id); err != nil {
		return nil, err
	}
	ids, err := s.AuthorizedMenuIds(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.TenantGetMenusRes{MenuIds: ids}, nil
}

// SetMenus 全量重写授权集 + casbin 双向联动 (同一事务, Reload 在提交后)。
func (s *sTenant) SetMenus(ctx context.Context, req *v1.TenantSetMenusReq) (res *v1.TenantSetMenusRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	if req.Id == consts.TenantPlatformId {
		return nil, xerror.New(xerror.CodeBusinessError, "平台租户授权集整体豁免, 无需配置")
	}
	if _, err = s.getTenant(ctx, req.Id); err != nil {
		return nil, err
	}
	menus, err := s.loadMenus(ctx)
	if err != nil {
		return nil, err
	}
	newSet := dedupU64(req.MenuIds)
	if verr := s.validateGrantSet(menus, newSet); verr != nil {
		return nil, verr
	}
	newClosure := closureOf(menus, newSet)
	newApiRows, err := s.allowedApis(ctx, newClosure)
	if err != nil {
		return nil, err
	}
	oldSet := make(map[uint64]struct{})
	if oldIds, oerr := s.AuthorizedMenuIds(ctx, req.Id); oerr == nil {
		for _, id := range oldIds {
			oldSet[id] = struct{}{}
		}
	}
	oldClosure := closureOf(menus, oldSet)

	dom := strconv.FormatUint(req.Id, 10)
	rid, terr := s.TenantAdminRoleId(ctx, req.Id)

	txErr := dao.SysTenantMenu.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 重写授权集 (PUT 全量覆盖, 幂等)
		if err := rewriteTenantMenus(ctx, req.Id, newSet); err != nil {
			return err
		}
		// 收缩: 删除域内越界 p 行 (菜单 + API), 对全部角色生效 (不变式强制)
		if err := shrinkDomainPolicies(ctx, dom, menus, newClosure, newApiRows); err != nil {
			return err
		}
		// 扩张: 仅 tenant_admin 角色追加新增菜单及对应 API 的 p 行 ("买了就用")
		if terr == nil && rid > 0 {
			oldAllowedApis, _ := s.allowedApis(ctx, oldClosure)
			if err := expandTenantAdminPolicies(ctx, rid, dom, menus, newClosure, oldClosure, newApiRows, oldAllowedApis); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, txErr, "设置应用授权集失败")
	}
	if _, rerr := casbinx.Reload(ctx); rerr != nil {
		g.Log().Warningf(ctx, "casbin reload after tenant menus change failed: %v", rerr)
	}
	return &v1.TenantSetMenusRes{}, nil
}

// ---------------------------------------------------------------------------
// 跨模块辅助
// ---------------------------------------------------------------------------

// AuthorizedMenuIds 授权集原始行 (目录/页面级)。
func (s *sTenant) AuthorizedMenuIds(ctx context.Context, tenantId uint64) ([]uint64, error) {
	// 平台视角查询 (调用方可能处于租户注入语境; sys_tenant_menu 不在注入白名单,
	// 显式 tenant_id 行本就按行内值过滤)
	qctx := contextx.WithoutTenant(ctx)
	res, err := dao.SysTenantMenu.Ctx(qctx).Where("tenant_id", tenantId).Fields("menu_id").All()
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(res))
	for _, r := range res {
		ids = append(ids, r["menu_id"].Uint64())
	}
	return ids, nil
}

// EffectiveMenuClosure 有效闭包 (授权行 ∪ 子孙 ∪ 祖先); 平台租户返回 nil (=全集豁免)。
func (s *sTenant) EffectiveMenuClosure(ctx context.Context, tenantId uint64) (map[uint64]struct{}, error) {
	if tenantId == consts.TenantPlatformId {
		return nil, nil
	}
	ids, err := s.AuthorizedMenuIds(ctx, tenantId)
	if err != nil {
		return nil, err
	}
	granted := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		granted[id] = struct{}{}
	}
	menus, err := s.loadMenus(contextx.WithoutTenant(ctx))
	if err != nil {
		return nil, err
	}
	return closureOf(menus, granted), nil
}

// TenantAdminRoleId 定位租户的 tenant_admin 克隆角色。
func (s *sTenant) TenantAdminRoleId(ctx context.Context, tenantId uint64) (uint64, error) {
	v, err := dao.SysRole.Ctx(contextx.WithoutTenant(ctx)).
		Where("tenant_id", tenantId).
		Where("code", consts.RoleTenantAdminCode).
		Where("deleted_at IS NULL").
		Fields("id").Value()
	if err != nil {
		return 0, err
	}
	if v == nil || v.IsNil() {
		return 0, gerrorNotFound
	}
	return v.Uint64(), nil
}

var gerrorNotFound = fmt.Errorf("tenant_admin role not found")

// ActiveTenantLites 全部启用且未到期的租户 (平台级任务逐租户巡检)。
func (s *sTenant) ActiveTenantLites(ctx context.Context) ([]*model.TenantLite, error) {
	return s.activeTenantRows(contextx.WithoutTenant(ctx))
}

func (s *sTenant) activeTenantRows(ctx context.Context) ([]*model.TenantLite, error) {
	var rows []*model.SysTenant
	if err := dao.SysTenant.Ctx(ctx).
		Where("status", consts.StatusEnabled).
		Where("deleted_at IS NULL").
		Order("id ASC").Scan(&rows); err != nil {
		return nil, err
	}
	now := gtime.Now()
	out := make([]*model.TenantLite, 0, len(rows))
	for _, r := range rows {
		if r.ExpireAt != nil && !r.ExpireAt.IsZero() && r.ExpireAt.Before(now) {
			continue
		}
		expire := ""
		if r.ExpireAt != nil && !r.ExpireAt.IsZero() {
			expire = r.ExpireAt.Format("Y-m-d H:i:s")
		}
		out = append(out, &model.TenantLite{Id: r.Id, Code: r.Code, Name: r.Name, Status: r.Status, ExpireAt: expire})
	}
	return out, nil
}

// LoadLiteById 按ID装载租户轻量行。
func (s *sTenant) LoadLiteById(ctx context.Context, id uint64) (*model.TenantLite, error) {
	var row *model.SysTenant
	if err := dao.SysTenant.Ctx(contextx.WithoutTenant(ctx)).
		Where("id", id).Where("deleted_at IS NULL").Scan(&row); err != nil {
		return nil, err
	}
	if row == nil {
		return nil, gerrorNotFound
	}
	return s.toLite(row), nil
}

// LoadLiteByCode 按租户码装载租户轻量行 (登录入口解析)。
func (s *sTenant) LoadLiteByCode(ctx context.Context, code string) (*model.TenantLite, error) {
	var row *model.SysTenant
	if err := dao.SysTenant.Ctx(contextx.WithoutTenant(ctx)).
		Where("code", code).Where("deleted_at IS NULL").Scan(&row); err != nil {
		return nil, err
	}
	if row == nil {
		return nil, gerrorNotFound
	}
	return s.toLite(row), nil
}

func (s *sTenant) toLite(r *model.SysTenant) *model.TenantLite {
	expire := ""
	if r.ExpireAt != nil && !r.ExpireAt.IsZero() {
		expire = r.ExpireAt.Format("Y-m-d H:i:s")
	}
	return &model.TenantLite{Id: r.Id, Code: r.Code, Name: r.Name, Status: r.Status, ExpireAt: expire}
}

// ---------------------------------------------------------------------------
// 内部工具
// ---------------------------------------------------------------------------

// loadMenus 全量菜单骨架 (B 类共享表, 平台视角)。
func (s *sTenant) loadMenus(ctx context.Context) ([]MenuLite, error) {
	res, err := dao.SysMenu.Ctx(contextx.WithoutTenant(ctx)).
		Where("deleted_at IS NULL").Where("status", consts.StatusEnabled).
		Fields("id, parent_id, type").All()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "加载菜单失败")
	}
	out := make([]MenuLite, 0, len(res))
	for _, r := range res {
		out = append(out, MenuLite{Id: r["id"].Uint64(), ParentId: r["parent_id"].Uint64(), Type: r["type"].Int()})
	}
	return out, nil
}

// validateGrantSet 授权集合法性: 仅目录/页面级 (按钮随父闭包) + 非平台专属。
func (s *sTenant) validateGrantSet(menus []MenuLite, set map[uint64]struct{}) error {
	byId := make(map[uint64]MenuLite, len(menus))
	for _, m := range menus {
		byId[m.Id] = m
	}
	for id := range set {
		m, ok := byId[id]
		if !ok {
			return xerror.New(xerror.CodeMenuNotInAuthSet, fmt.Sprintf("菜单 %d 不存在或未启用", id))
		}
		if m.Type == consts.MenuTypeButton {
			return xerror.New(xerror.CodeMenuNotInAuthSet, "授权集仅支持目录/页面级菜单 (按钮随父页面闭包)")
		}
		if isPlatformExclusive(menus, id) {
			return xerror.New(xerror.CodeMenuNotInAuthSet, "平台专属菜单不可授予租户 (监控/工具/目录维护/租户管理)")
		}
	}
	return nil
}

// allowedApis 授权闭包可绑的 API 集 (sys_api.menu_id ∈ closure)。
func (s *sTenant) allowedApis(ctx context.Context, closure map[uint64]struct{}) ([]gdb.Record, error) {
	if len(closure) == 0 {
		return nil, nil
	}
	ids := make([]uint64, 0, len(closure))
	for id := range closure {
		ids = append(ids, id)
	}
	res, err := dao.SysApi.Ctx(contextx.WithoutTenant(ctx)).
		Where("deleted_at IS NULL").
		WhereIn("menu_id", ids).
		Fields("path, method").All()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "加载 API 资源失败")
	}
	return res, nil
}

// rewriteTenantMenus 全量重写授权集行 (显式 tenant_id, 不走注入)。
func rewriteTenantMenus(ctx context.Context, tenantId uint64, set map[uint64]struct{}) error {
	if _, err := dao.SysTenantMenu.Ctx(ctx).Where("tenant_id", tenantId).Delete(); err != nil {
		return err
	}
	if len(set) == 0 {
		return nil
	}
	rows := make([]g.Map, 0, len(set))
	for id := range set {
		rows = append(rows, g.Map{"tenant_id": tenantId, "menu_id": id})
	}
	_, err := dao.SysTenantMenu.Ctx(ctx).Data(rows).Insert()
	return err
}

// insertRolePolicies 为角色生成全套 p 行 (菜单闭包 + 对应 API)。
func insertRolePolicies(ctx context.Context, roleId uint64, dom string, closure map[uint64]struct{}, apiRows []gdb.Record) error {
	rid := strconv.FormatUint(roleId, 10)
	rows := make([]g.Map, 0, len(closure)+len(apiRows))
	for id := range closure {
		rows = append(rows, g.Map{"ptype": "p", "v0": rid, "v1": fmt.Sprintf("menu:%d", id), "v2": "*", "v3": dom})
	}
	for _, a := range apiRows {
		rows = append(rows, g.Map{"ptype": "p", "v0": rid, "v1": a["path"].String(), "v2": a["method"].String(), "v3": dom})
	}
	if len(rows) > 0 {
		if _, err := dao.CasbinRule.Ctx(ctx).Data(rows).Insert(); err != nil {
			return err
		}
	}
	return nil
}

// shrinkDomainPolicies 收缩联动: 删除域内越界 p 行 (菜单不在新闭包 / API 不在可绑集)。
// 对全部角色生效 (不变式: 租户域内 API p 行 ⊆ 授权集对应 API 集)。
func shrinkDomainPolicies(ctx context.Context, dom string, menus []MenuLite, newClosure map[uint64]struct{}, newApiRows []gdb.Record) error {
	// 收集域内现存 p 行
	rows, err := dao.CasbinRule.Ctx(ctx).Where("ptype", "p").Where("v3", dom).Fields("id, v1, v2").All()
	if err != nil {
		return err
	}
	allowedAPI := make(map[string]struct{}, len(newApiRows))
	for _, a := range newApiRows {
		allowedAPI[a["path"].String()+"|"+a["method"].String()] = struct{}{}
	}
	delIds := make([]uint64, 0)
	for _, r := range rows {
		obj := r["v1"].String()
		if strings.HasPrefix(obj, "menu:") {
			id, perr := strconv.ParseUint(strings.TrimPrefix(obj, "menu:"), 10, 64)
			if perr != nil {
				delIds = append(delIds, r["id"].Uint64()) // 无法解析的脏数据一并清理
				continue
			}
			if _, ok := newClosure[id]; !ok {
				delIds = append(delIds, r["id"].Uint64())
			}
		} else {
			if _, ok := allowedAPI[obj+"|"+r["v2"].String()]; !ok {
				delIds = append(delIds, r["id"].Uint64())
			}
		}
	}
	if len(delIds) > 0 {
		if _, err := dao.CasbinRule.Ctx(ctx).WhereIn("id", delIds).Delete(); err != nil {
			return err
		}
	}
	return nil
}

// expandTenantAdminPolicies 扩张联动: 仅 tenant_admin 角色追加新增菜单与新增 API 的 p 行。
func expandTenantAdminPolicies(ctx context.Context, roleId uint64, dom string, menus []MenuLite,
	newClosure, oldClosure map[uint64]struct{}, newApiRows, oldApiRows []gdb.Record) error {

	// 角色现存 p 行 (去重依据)
	existing, err := dao.CasbinRule.Ctx(ctx).
		Where("ptype", "p").Where("v0", strconv.FormatUint(roleId, 10)).Where("v3", dom).
		Fields("v1, v2").All()
	if err != nil {
		return err
	}
	have := make(map[string]struct{}, len(existing))
	for _, r := range existing {
		have[r["v1"].String()+"|"+r["v2"].String()] = struct{}{}
	}

	rid := strconv.FormatUint(roleId, 10)
	var rows []g.Map
	// 新增菜单 (新闭包有、旧闭包无, 或角色缺行)
	for id := range newClosure {
		if _, old := oldClosure[id]; old {
			continue
		}
		key := fmt.Sprintf("menu:%d|*", id)
		if _, ok := have[key]; ok {
			continue
		}
		rows = append(rows, g.Map{"ptype": "p", "v0": rid, "v1": fmt.Sprintf("menu:%d", id), "v2": "*", "v3": dom})
	}
	// 新增 API (新可绑集有、旧集无)
	oldAPI := make(map[string]struct{}, len(oldApiRows))
	for _, a := range oldApiRows {
		oldAPI[a["path"].String()+"|"+a["method"].String()] = struct{}{}
	}
	for _, a := range newApiRows {
		key := a["path"].String() + "|" + a["method"].String()
		if _, old := oldAPI[key]; old {
			continue
		}
		if _, ok := have[key]; ok {
			continue
		}
		rows = append(rows, g.Map{"ptype": "p", "v0": rid, "v1": a["path"].String(), "v2": a["method"].String(), "v3": dom})
	}
	if len(rows) > 0 {
		if _, err := dao.CasbinRule.Ctx(ctx).Data(rows).Insert(); err != nil {
			return err
		}
	}
	return nil
}

// parseExpire 解析到期时间 (空返回 nil)。
func parseExpire(s string) *gtime.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if t, err := gtime.StrToTime(s); err == nil {
		return t
	}
	return nil
}

// dedupU64 去重。
func dedupU64(in []uint64) map[uint64]struct{} {
	out := make(map[uint64]struct{}, len(in))
	for _, v := range in {
		out[v] = struct{}{}
	}
	return out
}

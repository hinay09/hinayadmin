// tenantpackage.go —— 租户套餐: 应用授权集的模板复用层 (§7.2)。
//
// 设计要点:
//   - 套餐只是"目录/页面级菜单 ID 集"的命名快照 (menu_ids JSON 列, 升序稳定),
//     保存与同步均走与租户授权集相同的 validateGrantSet 校验
//     (仅目录/页面级 + 非平台专属; 菜单骨架漂移时同步显式报错而非静默丢弃);
//   - 复用姿势两种: 创建租户向导选中套餐预填勾选 (仍可微调, 显式 menuIds 优先),
//     或修改套餐后 Sync 显式物化到绑定租户 —— 走 SetMenus 同一路径
//     (applyMenuSet: 重写授权集 + casbin 双向联动), 逐租户独立事务、整体 Reload 一次;
//   - 修改套餐不自动触碰已绑定租户 (授权变更必须显式动作);
//   - 使用中 (有未删租户绑定) 禁删; 软删时事务内 name 改写 {name}#del{unix}
//     防同名校建撞 uk (同 sys_tenant.code 先例);
//   - 平台表无 tenant_id 列, 不进注入白名单; 入口双重守卫同租户管理
//     (multiTenant.enable + 平台超管 + 平台视角)。
package tenant

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

type sTenantPackage struct{}

func init() {
	service.RegisterTenantPackage(NewTenantPackage())
}

func NewTenantPackage() *sTenantPackage {
	return &sTenantPackage{}
}

// PackageItem 列表行。
type PackageItem struct {
	Id          uint64      `json:"id"`
	Name        string      `json:"name"`
	MenuCount   int         `json:"menuCount"`
	TenantCount int64       `json:"tenantCount"`
	Status      int         `json:"status"`
	Remark      string      `json:"remark"`
	CreatedAt   *gtime.Time `json:"createdAt"`
}

// PackageLite 启用套餐轻量行 (含授权集模板, 创建向导/授权抽屉预填用)。
type PackageLite struct {
	Id      uint64   `json:"id"`
	Name    string   `json:"name"`
	MenuIds []uint64 `json:"menuIds"`
}

// PackageDetailItem 详情行 (含解析后的授权集模板与绑定租户数)。
type PackageDetailItem struct {
	Id          uint64      `json:"id"`
	Name        string      `json:"name"`
	MenuIds     []uint64    `json:"menuIds"`
	Status      int         `json:"status"`
	Remark      string      `json:"remark"`
	TenantCount int64       `json:"tenantCount"`
	CreatedAt   *gtime.Time `json:"createdAt"`
	UpdatedAt   *gtime.Time `json:"updatedAt"`
}

// List 套餐分页列表 (含每套餐绑定租户数)。
func (s *sTenantPackage) List(ctx context.Context, req *v1.TenantPackageListReq) (res *v1.TenantPackageListRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	q := dao.SysTenantPackage.Ctx(ctx).Where("deleted_at IS NULL")
	if req.Keyword != "" {
		q = q.Where("name LIKE ?", "%"+strings.TrimSpace(req.Keyword)+"%")
	}
	if req.Status != nil {
		q = q.Where("status", *req.Status)
	}
	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "查询套餐失败")
	}
	var rows []*model.SysTenantPackage
	if err = q.Page(req.Page, req.PageSize).Order("id ASC").Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "查询套餐失败")
	}
	tCounts, err := tenantCountByPackage(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*PackageItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, &PackageItem{
			Id: r.Id, Name: r.Name, MenuCount: len(parseMenuIds(r.MenuIds)),
			TenantCount: tCounts[r.Id], Status: r.Status, Remark: r.Remark, CreatedAt: r.CreatedAt,
		})
	}
	page := response.Page(items, int64(total), req.Page, req.PageSize)
	out := v1.TenantPackageListRes(page)
	return &out, nil
}

// ListAll 启用套餐全量 (下拉预填)。
func (s *sTenantPackage) ListAll(ctx context.Context, req *v1.TenantPackageListAllReq) (res *v1.TenantPackageListAllRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	var rows []*model.SysTenantPackage
	if err = dao.SysTenantPackage.Ctx(ctx).
		Where("deleted_at IS NULL").Where("status", consts.StatusEnabled).
		Order("id ASC").Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "查询套餐失败")
	}
	list := make([]*PackageLite, 0, len(rows))
	for _, r := range rows {
		list = append(list, &PackageLite{Id: r.Id, Name: r.Name, MenuIds: parseMenuIds(r.MenuIds)})
	}
	return &v1.TenantPackageListAllRes{List: list}, nil
}

// Detail 套餐详情。
func (s *sTenantPackage) Detail(ctx context.Context, req *v1.TenantPackageDetailReq) (res *v1.TenantPackageDetailRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	row, err := getPackageRow(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	tCounts, err := tenantCountByPackage(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.TenantPackageDetailRes{Package: &PackageDetailItem{
		Id: row.Id, Name: row.Name, MenuIds: parseMenuIds(row.MenuIds), Status: row.Status,
		Remark: row.Remark, TenantCount: tCounts[row.Id], CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}}, nil
}

// Create 新增套餐。
func (s *sTenantPackage) Create(ctx context.Context, req *v1.TenantPackageCreateReq) (res *v1.TenantPackageCreateRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	if n, _ := dao.SysTenantPackage.Ctx(ctx).Where("name", req.Name).Where("deleted_at IS NULL").Count(); n > 0 {
		return nil, xerror.New(xerror.CodePackageNameExists)
	}
	set, verr := validateMenuSet(ctx, req.MenuIds)
	if verr != nil {
		return nil, verr
	}
	id, err := dao.SysTenantPackage.Ctx(ctx).Data(g.Map{
		"name": req.Name, "menu_ids": encodeMenuSet(set), "status": req.Status, "remark": req.Remark,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "新增套餐失败")
	}
	return &v1.TenantPackageCreateRes{Id: uint64(id)}, nil
}

// Update 修改套餐 (仅改模板, 不触碰已绑定租户的授权集 —— 生效走 Sync)。
func (s *sTenantPackage) Update(ctx context.Context, req *v1.TenantPackageUpdateReq) (res *v1.TenantPackageUpdateRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = getPackageRow(ctx, req.Id); err != nil {
		return nil, err
	}
	if n, _ := dao.SysTenantPackage.Ctx(ctx).Where("name", req.Name).
		Where("id != ?", req.Id).Where("deleted_at IS NULL").Count(); n > 0 {
		return nil, xerror.New(xerror.CodePackageNameExists)
	}
	set, verr := validateMenuSet(ctx, req.MenuIds)
	if verr != nil {
		return nil, verr
	}
	if _, err = dao.SysTenantPackage.Ctx(ctx).Where("id", req.Id).Data(g.Map{
		"name": req.Name, "menu_ids": encodeMenuSet(set), "status": req.Status, "remark": req.Remark,
	}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "修改套餐失败")
	}
	return &v1.TenantPackageUpdateRes{}, nil
}

// Delete 删除套餐 (使用中禁止; 软删 + 事务内 name 改写防同名校建撞 uk)。
func (s *sTenantPackage) Delete(ctx context.Context, req *v1.TenantPackageDeleteReq) (res *v1.TenantPackageDeleteRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	row, err := getPackageRow(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if n, cerr := dao.SysTenant.Ctx(ctx).Where("deleted_at IS NULL").Where("package_id", req.Id).Count(); cerr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, cerr, "查询套餐绑定租户失败")
	} else if n > 0 {
		return nil, xerror.New(xerror.CodePackageInUse, fmt.Sprintf("套餐使用中 (绑定 %d 个租户), 请先解绑或换绑后再删除", n))
	}
	newName := fmt.Sprintf("%s#del%d", row.Name, time.Now().Unix())
	txErr := dao.SysTenantPackage.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		_, err := dao.SysTenantPackage.Ctx(ctx).Where("id", req.Id).Data(g.Map{
			"deleted_at": gtime.Now(), "name": newName,
		}).Update()
		return err
	})
	if txErr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, txErr, "删除套餐失败")
	}
	return &v1.TenantPackageDeleteRes{}, nil
}

// Sync 套餐同步至全部绑定租户 (逐租户 SetMenus 同路径, 整体完成后 Reload 一次)。
func (s *sTenantPackage) Sync(ctx context.Context, req *v1.TenantPackageSyncReq) (res *v1.TenantPackageSyncRes, err error) {
	ctx, err = guard(ctx)
	if err != nil {
		return nil, err
	}
	row, err := getPackageRow(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	// 同步前整体校验: 菜单骨架漂移 (删除/停用/转平台专属) 时显式报错, 不静默丢项
	set, verr := validateMenuSet(ctx, parseMenuIds(row.MenuIds))
	if verr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, verr, "套餐授权集模板已失效, 请先修正套餐")
	}
	ids := sortedIds(set)
	tids, err := boundTenantIds(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	st := NewTenant()
	done := 0
	for _, tid := range tids {
		if err = st.applyMenuSet(ctx, tid, ids); err != nil {
			return nil, xerror.Wrap(xerror.CodeBusinessError, err, fmt.Sprintf("套餐同步失败: 租户 %d", tid))
		}
		done++
	}
	if _, rerr := casbinx.Reload(ctx); rerr != nil {
		g.Log().Warningf(ctx, "casbin reload after package sync failed: %v", rerr)
	}
	return &v1.TenantPackageSyncRes{Count: done}, nil
}

// ---------------------------------------------------------------------------
// 内部工具 (同包共用, 供 tenant.go 的创建/列表联动)
// ---------------------------------------------------------------------------

// getPackageRow 按 ID 取套餐行 (未删), 不存在报业务错误。
func getPackageRow(ctx context.Context, id uint64) (*model.SysTenantPackage, error) {
	var row *model.SysTenantPackage
	if err := dao.SysTenantPackage.Ctx(ctx).Where("id", id).Where("deleted_at IS NULL").Scan(&row); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "查询套餐失败")
	}
	if row == nil {
		return nil, xerror.New(xerror.CodeNotFound, "套餐不存在")
	}
	return row, nil
}

// packageNameMap 未删套餐 id→名称 (租户列表联表展示)。
func packageNameMap(ctx context.Context) map[uint64]string {
	res, err := dao.SysTenantPackage.Ctx(ctx).Where("deleted_at IS NULL").Fields("id, name").All()
	out := make(map[uint64]string, len(res))
	if err != nil {
		return out
	}
	for _, r := range res {
		out[r["id"].Uint64()] = r["name"].String()
	}
	return out
}

// tenantCountByPackage 每套餐绑定的未删租户数。
func tenantCountByPackage(ctx context.Context) (map[uint64]int64, error) {
	res, err := dao.SysTenant.Ctx(ctx).Where("deleted_at IS NULL").
		Where("package_id > 0").Fields("package_id, COUNT(*) AS c").
		Group("package_id").All()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "统计套餐租户失败")
	}
	out := make(map[uint64]int64, len(res))
	for _, r := range res {
		out[r["package_id"].Uint64()] = r["c"].Int64()
	}
	return out, nil
}

// boundTenantIds 绑定某套餐的未删租户 (id 升序; 平台租户不可能被绑定, 无需排除)。
func boundTenantIds(ctx context.Context, packageId uint64) ([]uint64, error) {
	res, err := dao.SysTenant.Ctx(ctx).Where("deleted_at IS NULL").
		Where("package_id", packageId).Fields("id").Order("id ASC").All()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "查询套餐绑定租户失败")
	}
	ids := make([]uint64, 0, len(res))
	for _, r := range res {
		ids = append(ids, r["id"].Uint64())
	}
	return ids, nil
}

// validateMenuSet 授权集模板校验 (与租户授权集同一套规则) + 去重, 返回可入库集合。
func validateMenuSet(ctx context.Context, ids []uint64) (map[uint64]struct{}, error) {
	st := NewTenant()
	menus, err := st.loadMenus(ctx)
	if err != nil {
		return nil, err
	}
	set := dedupU64(ids)
	if len(set) == 0 {
		return nil, xerror.New(xerror.CodeBusinessError, "套餐授权集不能为空")
	}
	if verr := st.validateGrantSet(menus, set); verr != nil {
		return nil, verr
	}
	return set, nil
}

// sortedIds 集合转升序切片 (快照编码/同步物化, 顺序稳定)。
func sortedIds(set map[uint64]struct{}) []uint64 {
	ids := make([]uint64, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// encodeMenuSet 集合编码为 menu_ids JSON (升序)。
func encodeMenuSet(set map[uint64]struct{}) string {
	return encodeMenuIds(sortedIds(set))
}

// encodeMenuIds / parseMenuIds menu_ids JSON 数组列编解码 (纯函数, 单测覆盖)。
func encodeMenuIds(ids []uint64) string {
	if len(ids) == 0 {
		return "[]"
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func parseMenuIds(s string) []uint64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var ids []uint64
	if err := json.Unmarshal([]byte(s), &ids); err != nil {
		return nil
	}
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			out = append(out, id)
		}
	}
	return out
}

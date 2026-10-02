// Package datascope 组织架构数据权限。
//
// 角色通过 data_scope 声明数据范围 (consts.DataScope*):
//
//	1=全部数据  2=自定义组织(sys_role_org)  3=本部门  4=本部门及以下  5=仅本人
//
// 用户拥有多个角色时取并集 (任一角色为"全部"则视为全部)。
// 业务查询通过 Apply 将范围条件套用到任意带组织字段的查询上;
// 范围为空 (无角色且无可见组织) 时套恒假条件, 语义为 fail-close。
package datascope

import (
	"context"
	"slices"

	"github.com/gogf/gf/v2/database/gdb"

	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/logic/casbinx"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/xerror"
)

type sDataScope struct{}

func init() {
	service.RegisterDataScope(NewDataScope())
}

func NewDataScope() *sDataScope {
	return &sDataScope{}
}

// OrgScope 计算当前登录用户的组织数据范围。
// admin 角色恒为全部 (与 Casbin 中间件的全局放行语义一致)。
func (s *sDataScope) OrgScope(ctx context.Context) (*model.OrgScope, error) {
	cur := contextx.LoginUser(ctx)
	if cur == nil {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	scope := &model.OrgScope{}

	// admin 全局放行
	if contextx.IsAdmin(ctx) {
		scope.All = true
		return scope, nil
	}

	// 角色ID列表 (g 行按用户ID关联)
	roleIds, err := casbinx.GetUserRoles(ctx, cur.UserId)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if len(roleIds) == 0 {
		// 无角色: 按仅本人处理 (基础会话可用, 但看不到任何他人数据)
		scope.Self = true
		return scope, nil
	}

	// 角色的数据范围配置
	var rows []*model.SysRole
	if err = dao.SysRole.Ctx(ctx).
		Fields("id, data_scope").
		WhereIn("id", roleIds).
		Where("deleted_at IS NULL").
		Where("status", consts.StatusEnabled).
		Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if len(rows) == 0 {
		scope.Self = true
		return scope, nil
	}

	// 用户所属组织
	var u *model.SysUser
	if err = dao.SysUser.Ctx(ctx).Fields("id, org_id").Where("id", cur.UserId).Scan(&u); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var userOrgId uint64
	if u != nil {
		userOrgId = u.OrgId
	}

	var customRoleIds []uint64
	for _, r := range rows {
		switch r.DataScope {
		case consts.DataScopeAll:
			scope.All = true
		case consts.DataScopeCustom:
			customRoleIds = append(customRoleIds, r.Id)
		case consts.DataScopeDept:
			if userOrgId > 0 {
				scope.OrgIds = appendUnique(scope.OrgIds, userOrgId)
			}
		case consts.DataScopeDeptSub:
			if userOrgId > 0 {
				ids, derr := s.descendantOrgIds(ctx, userOrgId)
				if derr != nil {
					return nil, derr
				}
				for _, id := range ids {
					scope.OrgIds = appendUnique(scope.OrgIds, id)
				}
			}
		case consts.DataScopeSelf:
			scope.Self = true
		}
		if scope.All {
			break
		}
	}

	// 自定义组织的 ID 集合
	if len(customRoleIds) > 0 {
		mappings, merr := dao.SysRoleOrg.Ctx(ctx).
			Fields("role_id, org_id").
			WhereIn("role_id", customRoleIds).
			All()
		if merr != nil {
			return nil, xerror.Wrap(xerror.CodeBusinessError, merr)
		}
		for _, m := range mappings {
			scope.OrgIds = appendUnique(scope.OrgIds, m["org_id"].Uint64())
		}
	}
	if scope.All {
		scope.OrgIds = nil
		scope.Self = false
	}
	return scope, nil
}

// Apply 将数据范围套用到查询 (orgColumn/selfColumn 只允许来自调用方代码,
// 不允许来自外部输入, 此处直接拼接进 SQL)。
func (s *sDataScope) Apply(ctx context.Context, m *gdb.Model, orgColumn, selfColumn string) (*gdb.Model, error) {
	scope, err := s.OrgScope(ctx)
	if err != nil {
		return nil, err
	}
	cur := contextx.LoginUser(ctx)
	switch {
	case scope.All:
		return m, nil
	case scope.Self && len(scope.OrgIds) > 0 && cur != nil:
		// (self=uid OR org IN (...)) —— 括号保证与外层其他 AND 条件正确组合
		return m.Where(
			"("+selfColumn+" = ? OR "+orgColumn+" IN(?))",
			cur.UserId, scope.OrgIds,
		), nil
	case scope.Self && cur != nil:
		return m.Where(selfColumn, cur.UserId), nil
	case len(scope.OrgIds) > 0:
		return m.WhereIn(orgColumn, scope.OrgIds), nil
	default:
		// 无任何可见范围: 恒假, fail-close
		return m.Where("1=0"), nil
	}
}

// descendantOrgIds 收集组织 ID 的全部后代 (含自身), 内存 BFS。
func (s *sDataScope) descendantOrgIds(ctx context.Context, rootId uint64) ([]uint64, error) {
	all, err := dao.SysOrg.Ctx(ctx).Fields("id, parent_id").Where("deleted_at IS NULL").All()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	children := make(map[uint64][]uint64)
	for _, row := range all {
		pid := row["parent_id"].Uint64()
		children[pid] = append(children[pid], row["id"].Uint64())
	}
	result := []uint64{rootId}
	queue := []uint64{rootId}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, cid := range children[cur] {
			result = append(result, cid)
			queue = append(queue, cid)
		}
	}
	return result, nil
}

// appendUnique 向集合追加去重 ID。
func appendUnique(ids []uint64, id uint64) []uint64 {
	if slices.Contains(ids, id) {
		return ids
	}
	return append(ids, id)
}

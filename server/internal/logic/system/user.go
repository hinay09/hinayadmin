// Package system 系统管理-用户业务逻辑。
package system

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/logic/casbinx"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/password"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

type sUser struct{}

func NewUser() *sUser {
	return &sUser{}
}

func init() {
	service.RegisterUser(NewUser())
}

// List 分页列表。
func (s *sUser) List(ctx context.Context, req *v1.UserListReq) (res *v1.UserListRes, err error) {
	q := dao.SysUser.Ctx(ctx).Where("deleted_at IS NULL")
	if req.Keyword != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		q = q.WhereOr("username LIKE ?", kw).WhereOr("nickname LIKE ?", kw)
	}
	if req.Status != nil {
		q = q.Where("status", *req.Status)
	}
	total, err := q.Ctx(ctx).Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	var rows []*model.SysUser
	if err = q.Ctx(ctx).
		Page(req.Page, req.PageSize).
		Order("id DESC").
		Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}

	// 从 Casbin 加载角色
	list := make([]*v1.UserVO, 0, len(rows))
	for _, u := range rows {
		roles, _ := casbinx.GetUserRoles(ctx, u.Username)
		orgName := getOrgName(ctx, u.OrgId)
		list = append(list, &v1.UserVO{
			Id:        u.Id,
			Username:  u.Username,
			Nickname:  u.Nickname,
			Avatar:    u.Avatar,
			Email:     u.Email,
			Phone:     u.Phone,
			OrgId:     u.OrgId,
			OrgName:   orgName,
			Status:    u.Status,
			Remark:    u.Remark,
			Roles:     roles,
			CreatedAt: u.CreatedAt,
		})
	}
	page := response.Page(list, int64(total), req.Page, req.PageSize)
	r := v1.UserListRes(page)
	return &r, nil
}

// Detail 详情。
func (s *sUser) Detail(ctx context.Context, req *v1.UserDetailReq) (res *v1.UserDetailRes, err error) {
	var u *model.SysUser
	if err = dao.SysUser.Ctx(ctx).Where("id", req.Id).Where("deleted_at IS NULL").Scan(&u); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if u == nil {
		return nil, xerror.New(xerror.CodeUserNotFound)
	}
	roles, _ := casbinx.GetUserRoles(ctx, u.Username)
	orgName := getOrgName(ctx, u.OrgId)
	return &v1.UserDetailRes{UserVO: &v1.UserVO{
		Id:        u.Id,
		Username:  u.Username,
		Nickname:  u.Nickname,
		Avatar:    u.Avatar,
		Email:     u.Email,
		Phone:     u.Phone,
		OrgId:     u.OrgId,
		OrgName:   orgName,
		Status:    u.Status,
		Remark:    u.Remark,
		Roles:     roles,
		CreatedAt: u.CreatedAt,
	}}, nil
}

// Create 新增。
func (s *sUser) Create(ctx context.Context, req *v1.UserCreateReq) (res *v1.UserCreateRes, err error) {
	cnt, _ := dao.SysUser.Ctx(ctx).Where("username", req.Username).Where("deleted_at IS NULL").Count()
	if cnt > 0 {
		return nil, xerror.New(xerror.CodeUsernameExists)
	}
	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if req.Status == 0 {
		req.Status = consts.StatusEnabled
	}
	id, err := dao.SysUser.Ctx(ctx).Data(g.Map{
		"username": req.Username,
		"password": hash,
		"nickname": req.Nickname,
		"email":    req.Email,
		"phone":    req.Phone,
		"org_id":   req.OrgId,
		"status":   req.Status,
		"remark":   req.Remark,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	// 分配角色: RoleIds -> 查询角色 code -> casbinx.SetUserRoles
	// 任一环节失败都需要回滚已创建的用户行, 避免出现"用户已建但无角色"的脏数据。
	if len(req.RoleIds) > 0 {
		roleCodes, rerr := loadRoleCodesByIds(ctx, req.RoleIds)
		if rerr != nil {
			g.Log().Errorf(ctx, "UserCreate loadRoleCodesByIds failed, roleIds=%v err=%v", req.RoleIds, rerr)
			rollbackUser(ctx, id)
			return nil, xerror.Wrap(xerror.CodeBusinessError, rerr)
		}
		if len(roleCodes) != len(req.RoleIds) {
			g.Log().Warningf(ctx, "UserCreate role mismatch, roleIds=%v codes=%v", req.RoleIds, roleCodes)
		}
		if len(roleCodes) == 0 {
			rollbackUser(ctx, id)
			return nil, xerror.New(xerror.CodeBusinessError, "指定的角色不存在或已被删除")
		}
		if serr := casbinx.SetUserRoles(ctx, req.Username, roleCodes); serr != nil {
			g.Log().Errorf(ctx, "UserCreate SetUserRoles failed, username=%s codes=%v err=%v", req.Username, roleCodes, serr)
			rollbackUser(ctx, id)
			return nil, xerror.Wrap(xerror.CodeBusinessError, serr)
		}
	}
	return &v1.UserCreateRes{Id: uint64(id)}, nil
}

// rollbackUser 物理删除刚插入的用户, 用于角色绑定失败时的补偿。
func rollbackUser(ctx context.Context, id int64) {
	if _, err := dao.SysUser.Ctx(ctx).Where("id", id).Delete(); err != nil {
		g.Log().Errorf(ctx, "rollbackUser failed, id=%d err=%v", id, err)
	}
}

// Update 修改。
func (s *sUser) Update(ctx context.Context, req *v1.UserUpdateReq) (res *v1.UserUpdateRes, err error) {
	// 内置 admin (id=1) 保护: 不允许禁用, 不允许解绑 admin 角色
	if req.Id == 1 {
		if req.Status != consts.StatusEnabled {
			return nil, xerror.New(xerror.CodeBusinessError, "内置管理员不可禁用")
		}
		if req.RoleIds != nil {
			roleCodes, lerr := loadRoleCodesByIds(ctx, req.RoleIds)
			if lerr != nil {
				return nil, xerror.Wrap(xerror.CodeBusinessError, lerr)
			}
			hasAdmin := false
			for _, c := range roleCodes {
				if c == consts.RoleAdmin {
					hasAdmin = true
					break
				}
			}
			if !hasAdmin {
				return nil, xerror.New(xerror.CodeBusinessError, "内置管理员不可解绑超管角色")
			}
		}
	}
	if _, err = dao.SysUser.Ctx(ctx).
		Where("id", req.Id).Where("deleted_at IS NULL").Ctx(ctx).
		Data(g.Map{
			"nickname": req.Nickname,
			"email":    req.Email,
			"phone":    req.Phone,
			"org_id":   req.OrgId,
			"status":   req.Status,
			"remark":   req.Remark,
		}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if req.RoleIds != nil {
		unameVal, verr := dao.SysUser.Ctx(ctx).Where("id", req.Id).Fields("username").Value()
		if verr != nil {
			g.Log().Errorf(ctx, "UserUpdate query username failed, id=%d err=%v", req.Id, verr)
			return nil, xerror.Wrap(xerror.CodeBusinessError, verr)
		}
		username := strings.TrimSpace(unameVal.String())
		if username == "" {
			return nil, xerror.New(xerror.CodeUserNotFound)
		}
		roleCodes, rerr := loadRoleCodesByIds(ctx, req.RoleIds)
		if rerr != nil {
			g.Log().Errorf(ctx, "UserUpdate loadRoleCodesByIds failed, roleIds=%v err=%v", req.RoleIds, rerr)
			return nil, xerror.Wrap(xerror.CodeBusinessError, rerr)
		}
		if serr := casbinx.SetUserRoles(ctx, username, roleCodes); serr != nil {
			g.Log().Errorf(ctx, "UserUpdate SetUserRoles failed, username=%s codes=%v err=%v", username, roleCodes, serr)
			return nil, xerror.Wrap(xerror.CodeBusinessError, serr)
		}
	}
	return &v1.UserUpdateRes{}, nil
}

// Delete 软删除 + 清理 Casbin g 策略。
func (s *sUser) Delete(ctx context.Context, req *v1.UserDeleteReq) (res *v1.UserDeleteRes, err error) {
	if req.Id == 1 {
		return nil, xerror.New(xerror.CodeBusinessError, "内置管理员不可删除")
	}
	// 查出用户 username, 用于清理 Casbin g 策略
	unameVal, _ := dao.SysUser.Ctx(ctx).Where("id", req.Id).Fields("username").Value()
	username := strings.TrimSpace(unameVal.String())

	if _, err = dao.SysUser.Ctx(ctx).Where("id", req.Id).
		Data(g.Map{"deleted_at": gtime.Now()}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	// 清理 Casbin g 策略中该用户的角色映射
	if username != "" {
		_ = casbinx.SetUserRoles(ctx, username, nil)
	}
	return &v1.UserDeleteRes{}, nil
}

// ResetPwd 重置密码。
func (s *sUser) ResetPwd(ctx context.Context, req *v1.UserResetPwdReq) (res *v1.UserResetPwdRes, err error) {
	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if _, err = dao.SysUser.Ctx(ctx).Where("id", req.Id).
		Data(g.Map{"password": hash}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.UserResetPwdRes{}, nil
}

// loadRoleCodesByIds 根据 role ID 列表查询角色 code 列表。
func loadRoleCodesByIds(ctx context.Context, ids []uint64) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	values, err := dao.SysRole.Ctx(ctx).
		WhereIn("id", ids).
		Where("deleted_at IS NULL").
		Fields("code").
		Ctx(ctx).Array()
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(values))
	for _, v := range values {
		if str := strings.TrimSpace(v.String()); str != "" {
			codes = append(codes, str)
		}
	}
	return codes, nil
}

// getOrgName 根据组织ID获取组织名称, 不存在或 orgId=0 时返回空串。
func getOrgName(ctx context.Context, orgId uint64) string {
	if orgId == 0 {
		return ""
	}
	val, err := dao.SysOrg.Ctx(ctx).
		Where("id", orgId).Where("deleted_at IS NULL").
		Fields("name").Value()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(val.String())
}

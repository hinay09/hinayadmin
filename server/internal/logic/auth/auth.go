// Package auth 登录/登出/当前用户与菜单。
package auth

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/guid"
	"hinay.cn/admin/internal/dao"

	v1 "hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/logic/casbinx"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/jwtx"
	"hinay.cn/admin/utility/password"
	"hinay.cn/admin/utility/xerror"
)

type sAuth struct{}

func init() {
	service.RegisterAuth(NewAuth())
}

func NewAuth() *sAuth {
	return &sAuth{}
}

// dedupUint64 对 uint64 切片去重并保持顺序。
func dedupUint64(in []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(in))
	out := make([]uint64, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// Login 用户名密码登录。
func (s *sAuth) Login(ctx context.Context, req *v1.LoginReq) (res *v1.LoginRes, err error) {
	var u *model.SysUser
	err = dao.SysUser.Ctx(ctx).
		Where("username", req.Username).
		Where("deleted_at IS NULL").
		Ctx(ctx).Scan(&u)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "查询用户失败")
	}
	if u == nil {
		return nil, xerror.New(xerror.CodePasswordWrong)
	}
	if u.Status != consts.StatusEnabled {
		return nil, xerror.New(xerror.CodeUserDisabled)
	}
	if !password.Verify(u.Password, req.Password) {
		return nil, xerror.New(xerror.CodePasswordWrong)
	}

	token, exp, err := jwtx.Generate(ctx, u.Id, u.Username)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "签发token失败")
	}
	roles, _ := casbinx.GetUserRoles(ctx, u.Username)
	return &v1.LoginRes{
		Token:    token,
		ExpireAt: exp,
		UserInfo: &model.LoginUser{
			UserId:   u.Id,
			Username: u.Username,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
			Roles:    roles,
		},
	}, nil
}

// Refresh 使用当前有效 token 续签新 token, 旧 token 加入黑名单。
func (s *sAuth) Refresh(ctx context.Context, req *v1.RefreshReq) (res *v1.RefreshRes, err error) {
	cur := contextx.LoginUser(ctx)
	if cur == nil {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	// 二次确认用户仍有效 (未被禁用/删除)
	var u *model.SysUser
	if err = dao.SysUser.Ctx(ctx).
		Where("id", cur.UserId).
		Where("status", consts.StatusEnabled).
		Where("deleted_at IS NULL").
		Ctx(ctx).Scan(&u); err != nil || u == nil {
		return nil, xerror.New(xerror.CodeUserNotFound)
	}

	// 旧 token 加入黑名单
	oldToken := contextx.JwtToken(ctx)
	if oldToken != "" {
		expire := time.Hour * 24
		if v, cerr := g.Cfg().Get(ctx, "jwt.expireSec"); cerr == nil {
			expire = time.Duration(v.Int64()) * time.Second
		}
		_, _ = g.Redis().Set(ctx, consts.JWTBlacklistPrefix+oldToken, 1)
		_, _ = g.Redis().Expire(ctx, consts.JWTBlacklistPrefix+oldToken, int64(expire.Seconds()))
	}

	token, exp, err := jwtx.Generate(ctx, u.Id, u.Username)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "签发token失败")
	}
	return &v1.RefreshRes{Token: token, ExpireAt: exp}, nil
}

// Logout 写入 Redis 黑名单。
func (s *sAuth) Logout(ctx context.Context, req *v1.LogoutReq) (res *v1.LogoutRes, err error) {
	token := contextx.JwtToken(ctx)
	if token == "" {
		return &v1.LogoutRes{}, nil
	}
	// 设置过期时间略大于 jwt 过期, 这里直接 24h
	expire := time.Hour * 24
	if v, cerr := g.Cfg().Get(ctx, "jwt.expireSec"); cerr == nil {
		expire = time.Duration(v.Int64()) * time.Second
	}
	if _, err = g.Redis().Set(ctx, consts.JWTBlacklistPrefix+token, 1); err != nil {
		return nil, err
	}
	_, _ = g.Redis().Expire(ctx, consts.JWTBlacklistPrefix+token, int64(expire.Seconds()))
	return &v1.LogoutRes{}, nil
}

// loadLoginUser 装载当前登录用户的最新信息。
func (s *sAuth) loadLoginUser(ctx context.Context) (*model.LoginUser, error) {
	cur := contextx.LoginUser(ctx)
	if cur == nil {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	var u *model.SysUser
	err := dao.SysUser.Ctx(ctx).
		Where("id", cur.UserId).
		Where("deleted_at IS NULL").
		Ctx(ctx).Scan(&u)
	if err != nil || u == nil {
		return nil, xerror.New(xerror.CodeUserNotFound)
	}
	roles, _ := casbinx.GetUserRoles(ctx, u.Username)
	return &model.LoginUser{
		UserId:   u.Id,
		Username: u.Username,
		Nickname: u.Nickname,
		Avatar:   u.Avatar,
		Email:    u.Email,
		Phone:    u.Phone,
		Roles:    roles,
	}, nil
}

// UserInfo 当前登录用户信息。
func (s *sAuth) UserInfo(ctx context.Context, req *v1.UserInfoReq) (res *v1.UserInfoRes, err error) {
	u, err := s.loadLoginUser(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.UserInfoRes{LoginUser: u}, nil
}

// Profile 个人中心-获取详情。
func (s *sAuth) Profile(ctx context.Context, req *v1.ProfileReq) (res *v1.ProfileRes, err error) {
	u, err := s.loadLoginUser(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.ProfileRes{LoginUser: u}, nil
}

// UpdateProfile 修改当前用户基础信息 (昵称/头像/邮箱/手机)。
func (s *sAuth) UpdateProfile(ctx context.Context, req *v1.UpdateProfileReq) (res *v1.UpdateProfileRes, err error) {
	cur := contextx.LoginUser(ctx)
	if cur == nil {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	if _, err = dao.SysUser.Ctx(ctx).
		Where("id", cur.UserId).
		Where("deleted_at IS NULL").
		Ctx(ctx).Data(g.Map{
		"nickname": req.Nickname,
		"avatar":   req.Avatar,
		"email":    req.Email,
		"phone":    req.Phone,
	}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "修改个人资料失败")
	}
	return &v1.UpdateProfileRes{}, nil
}

// ChangePassword 修改当前用户密码 (原密码校验 + bcrypt 重新生成)。
func (s *sAuth) ChangePassword(ctx context.Context, req *v1.ChangePasswordReq) (res *v1.ChangePasswordRes, err error) {
	cur := contextx.LoginUser(ctx)
	if cur == nil {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	if req.OldPassword == req.NewPassword {
		return nil, xerror.New(xerror.CodeBusinessError, "新密码不能与原密码相同")
	}
	var u *model.SysUser
	if err = dao.SysUser.Ctx(ctx).
		Where("id", cur.UserId).
		Where("deleted_at IS NULL").
		Ctx(ctx).Scan(&u); err != nil || u == nil {
		return nil, xerror.New(xerror.CodeUserNotFound)
	}
	if !password.Verify(u.Password, req.OldPassword) {
		return nil, xerror.New(xerror.CodeBusinessError, "原密码不正确")
	}
	hash, herr := password.Hash(req.NewPassword)
	if herr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, herr, "密码加密失败")
	}
	if _, err = dao.SysUser.Ctx(ctx).Where("id", cur.UserId).
		Data(g.Map{"password": hash}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "修改密码失败")
	}
	return &v1.ChangePasswordRes{}, nil
}

// avatarUploadDir 头像存储目录。
const avatarUploadDir = "resource/upload/avatar"

// allowedAvatarTypes 允许的头像 MIME 类型。
var allowedAvatarTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// UploadAvatar 上传并更新当前用户头像。
func (s *sAuth) UploadAvatar(ctx context.Context, req *v1.UploadAvatarReq) (res *v1.UploadAvatarRes, err error) {
	cur := contextx.LoginUser(ctx)
	if cur == nil {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}

	file := req.File
	// 校验文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	mime := file.Header.Get("Content-Type")
	if !allowedAvatarTypes[mime] {
		return nil, xerror.New(xerror.CodeBusinessError, "仅支持 JPG/PNG/GIF/WEBP 格式")
	}
	// 校验文件大小 (2MB)
	if file.Size > 2*1024*1024 {
		return nil, xerror.New(xerror.CodeBusinessError, "头像大小不能超过 2MB")
	}

	// 保存文件
	if err = os.MkdirAll(avatarUploadDir, 0755); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "创建目录失败")
	}
	saveName := fmt.Sprintf("%s%s", guid.S(), ext)
	savePath := filepath.Join(avatarUploadDir, saveName)

	src, err := file.Open()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "读取文件失败")
	}
	defer src.Close()

	dst, err := os.Create(savePath)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "保存文件失败")
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "写入文件失败")
	}

	avatarURL := fmt.Sprintf("/upload/avatar/%s", saveName)

	// 更新数据库
	if _, err = dao.SysUser.Ctx(ctx).
		Where("id", cur.UserId).
		Where("deleted_at IS NULL").
		Ctx(ctx).Data(g.Map{"avatar": avatarURL}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "更新头像失败")
	}

	return &v1.UploadAvatarRes{Url: avatarURL}, nil
}

// MenuTree 当前用户可见菜单树 (排除按钮类型)。
func (s *sAuth) MenuTree(ctx context.Context, req *v1.MenuTreeReq) (res *v1.MenuTreeRes, err error) {
	cur := contextx.LoginUser(ctx)
	if cur == nil {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}

	var menuIds []uint64

	if contextx.IsAdmin(ctx) {
		// 超管: 返回所有启用的可见菜单
		values, _ := dao.SysMenu.Ctx(ctx).
			Where("status", consts.StatusEnabled).
			Where("deleted_at IS NULL").
			Fields("id").Ctx(ctx).Array()
		for _, v := range values {
			menuIds = append(menuIds, v.Uint64())
		}
	} else {
		// 非超管: 通过 Casbin 获取角色 -> 获取菜单ID
		roles, rerr := casbinx.GetUserRoles(ctx, cur.Username)
		if rerr != nil {
			return nil, xerror.Wrap(xerror.CodeBusinessError, rerr, "获取用户角色失败")
		}
		seen := make(map[int64]struct{})
		for _, roleCode := range roles {
			ids, _ := casbinx.GetRoleMenus(ctx, roleCode)
			for _, id := range ids {
				if _, ok := seen[id]; !ok {
					seen[id] = struct{}{}
					menuIds = append(menuIds, uint64(id))
				}
			}
		}
	}

	if len(menuIds) == 0 {
		return &v1.MenuTreeRes{Menus: []*model.MenuTree{}, Permissions: []string{}}, nil
	}

	menuIds = dedupUint64(menuIds)

	// 菜单树: 仅可见且非按钮类型。
	var menus []*model.SysMenu
	if err = dao.SysMenu.Ctx(ctx).
		Where("status", consts.StatusEnabled).
		Where("visible", 1).
		Where("deleted_at IS NULL").
		WhereIn("id", menuIds).
		Order("sort ASC, id ASC").Ctx(ctx).Scan(&menus); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "加载菜单失败")
	}

	// 权限码收集: 包含按钮型菜单的 permission, 不受 visible 过滤。
	var permRows []*model.SysMenu
	if perr := dao.SysMenu.Ctx(ctx).
		Where("status", consts.StatusEnabled).
		Where("deleted_at IS NULL").
		Where("permission != ?", "").
		WhereIn("id", menuIds).
		Fields("permission").Ctx(ctx).Scan(&permRows); perr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, perr, "加载权限标识失败")
	}
	seenPerm := make(map[string]struct{}, len(permRows))
	perms := make([]string, 0, len(permRows))
	for _, m := range permRows {
		if m.Permission == "" {
			continue
		}
		if _, ok := seenPerm[m.Permission]; ok {
			continue
		}
		seenPerm[m.Permission] = struct{}{}
		perms = append(perms, m.Permission)
	}

	tree := buildTree(menus, 0)
	return &v1.MenuTreeRes{Menus: tree, Permissions: perms}, nil
}

func buildTree(list []*model.SysMenu, parentId uint64) []*model.MenuTree {
	out := make([]*model.MenuTree, 0)
	for _, m := range list {
		if m.ParentId == parentId && m.Type != consts.MenuTypeButton {
			node := &model.MenuTree{SysMenu: *m}
			node.Children = buildTree(list, m.Id)
			out = append(out, node)
		}
	}
	return out
}

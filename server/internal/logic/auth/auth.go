// Package auth 登录/登出/当前用户与菜单。
package auth

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	"hinay.cn/admin/utility/mimeutil"
	"hinay.cn/admin/utility/password"
	"hinay.cn/admin/utility/rsax"
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

// luaGetDel 原子"取出并删除"脚本, 保证一次性私钥不可被重复使用。
const luaGetDel = `local v = redis.call('GET', KEYS[1])
if v then redis.call('DEL', KEYS[1]) end
return v`

// PublicKey 生成一次性登录加密公钥。
// 每次调用生成全新 RSA 密钥对, 私钥存 Redis 并设置 TTL, 用后即毁;
// 按 IP 限流防止匿名端点被刷导致密钥生成 DoS。
func (s *sAuth) PublicKey(ctx context.Context, req *v1.PublicKeyReq) (res *v1.PublicKeyRes, err error) {
	// 单 IP 每分钟限流
	limitKey := consts.PubKeyLimitPrefix + clientIp(ctx)
	if n, lerr := g.Redis().Do(ctx, "INCR", limitKey); lerr == nil && n != nil && n.Int64() == 1 {
		_, _ = g.Redis().Do(ctx, "EXPIRE", limitKey, 60)
	} else if n != nil && n.Int64() > consts.PubKeyMaxPerMin {
		return nil, xerror.New(xerror.CodeBusinessError, "请求过于频繁, 请稍后再试")
	}

	kp, err := rsax.Generate()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "生成加密密钥失败")
	}
	if _, err = g.Redis().Do(ctx, "SET", consts.RSAKeyPrefix+kp.KeyId, kp.PrivatePem, "EX", consts.RSAKeyTTLSec); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "保存加密密钥失败")
	}
	return &v1.PublicKeyRes{KeyId: kp.KeyId, PublicKey: kp.PublicPem}, nil
}

// decryptPassword 用一次性私钥解密登录密码密文。
// 私钥取出即删除(EVAL 原子操作), 同一 keyId 无法二次使用。
func decryptPassword(ctx context.Context, keyId, cipher string) (string, error) {
	priv, err := g.Redis().Do(ctx, "EVAL", luaGetDel, 1, consts.RSAKeyPrefix+keyId)
	if err != nil {
		return "", xerror.Wrap(xerror.CodeBusinessError, err, "读取加密密钥失败")
	}
	if priv == nil || priv.IsEmpty() {
		return "", xerror.New(xerror.CodeRsaKeyInvalid)
	}
	plain, derr := rsax.Decrypt(priv.String(), cipher)
	if derr != nil {
		return "", xerror.New(xerror.CodeRsaKeyInvalid)
	}
	return plain, nil
}

// Login 用户名密码登录。
// 密码为前端用一次性公钥加密的 RSA 密文, 服务端解密后再走 bcrypt 校验。
// 带 IP+用户名 双维度失败计数防暴力破解: 窗口内失败超过 consts.LoginFailMax 次后临时锁定。
func (s *sAuth) Login(ctx context.Context, req *v1.LoginReq) (res *v1.LoginRes, err error) {
	failKey := consts.LoginFailPrefix + clientIp(ctx) + ":" + req.Username
	// 锁定检查: 窗口内失败次数已达上限
	if n, rerr := g.Redis().Do(ctx, "GET", failKey); rerr == nil && n != nil && n.Int64() >= consts.LoginFailMax {
		return nil, xerror.New(xerror.CodeBusinessError, "失败次数过多, 请 15 分钟后再试")
	}

	// 解密密码(一次性私钥), 解密失败不计入密码错误次数
	plainPassword, derr := decryptPassword(ctx, req.KeyId, req.Password)
	if derr != nil {
		return nil, derr
	}
	if len(plainPassword) < 6 || len(plainPassword) > 32 {
		return nil, xerror.New(xerror.CodeParamInvalid, "密码长度 6-32")
	}

	var u *model.SysUser
	err = dao.SysUser.Ctx(ctx).
		Where("username", req.Username).
		Where("deleted_at IS NULL").
		Ctx(ctx).Scan(&u)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "查询用户失败")
	}
	if u == nil {
		// 与"密码错误"走相同 bcrypt 计算与错误码, 抹平时间差与提示差异, 防用户名枚举
		password.Verify(dummyBcryptHash, plainPassword)
		recordLoginFailure(ctx, failKey)
		return nil, xerror.New(xerror.CodePasswordWrong)
	}
	if u.Status != consts.StatusEnabled {
		return nil, xerror.New(xerror.CodeUserDisabled)
	}
	if !password.Verify(u.Password, plainPassword) {
		recordLoginFailure(ctx, failKey)
		return nil, xerror.New(xerror.CodePasswordWrong)
	}

	// 登录成功, 清除失败计数
	_, _ = g.Redis().Do(ctx, "DEL", failKey)

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

// dummyBcryptHash 任意随机明文的合法 bcrypt 哈希, 仅用于用户不存在时制造等时比较。
const dummyBcryptHash = "$2a$10$7EqJtq98hPqEX7fNZaFWoOhi5B0G1S3kA9dQ5tqKqQ9wYvJmQ3kOu"

// clientIp 从请求上下文提取客户端 IP。
func clientIp(ctx context.Context) string {
	if r := g.RequestFromCtx(ctx); r != nil {
		return r.GetClientIp()
	}
	return "unknown"
}

// recordLoginFailure 累计一次登录失败; 首次失败时设置统计窗口 TTL。
func recordLoginFailure(ctx context.Context, failKey string) {
	n, err := g.Redis().Do(ctx, "INCR", failKey)
	if err != nil {
		return
	}
	if n != nil && n.Int64() == 1 {
		_, _ = g.Redis().Do(ctx, "EXPIRE", failKey, consts.LoginFailWindowSec)
	}
}

// blacklistToken 将 token 原子写入黑名单, TTL 覆盖 token 剩余生命周期。
func blacklistToken(ctx context.Context, token string) error {
	ttl := int64(86400)
	if v, cerr := g.Cfg().Get(ctx, "jwt.expireSec"); cerr == nil && v.Int64() > 0 {
		ttl = v.Int64()
	}
	_, err := g.Redis().Do(ctx, "SET", consts.JWTBlacklistPrefix+token, 1, "EX", ttl)
	return err
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
		if berr := blacklistToken(ctx, oldToken); berr != nil {
			return nil, xerror.Wrap(xerror.CodeBusinessError, berr, "旧token注销失败")
		}
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
	if err = blacklistToken(ctx, token); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "登出失败")
	}
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

// allowedAvatarExts 允许的头像扩展名(不含 svg: svg 可内嵌脚本, 是存储型 XSS 载体)。
var allowedAvatarExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
}

// allowedAvatarTypes 允许的、由服务端内容嗅探得到的 MIME 类型。
// multipart 的 Content-Type 头完全由客户端控制, 不可作为校验依据。
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
	// 扩展名白名单
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedAvatarExts[ext] {
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

	// 内容嗅探: 按文件头识别真实类型, 拒绝伪造扩展名/伪造 Content-Type 的文件
	head := make([]byte, 512)
	n, _ := src.Read(head)
	if detected := mimeutil.Detect(head[:n]); !allowedAvatarTypes[detected] {
		return nil, xerror.New(xerror.CodeBusinessError, "文件内容不是有效的图片")
	}
	if _, serr := src.Seek(0, io.SeekStart); serr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, serr, "读取文件失败")
	}

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

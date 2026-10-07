// 注册: 受全局配置 sys.allow_register 开关控制的自助开户。
// 与登录共用一次性 RSA 密钥加密通道, 注册成功不自动签发 token,
// 引导用户走完整登录流程 (含两步验证)。
package auth

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/logic/casbinx"
	"hinay.cn/admin/internal/logic/pwdpolicy"
	"hinay.cn/admin/utility/demox"
	"hinay.cn/admin/utility/password"
	"hinay.cn/admin/utility/xerror"
)

// KeyAllowRegister 开放注册开关键 (与 init.sql sys_config 种子对齐)。
const KeyAllowRegister = "sys.allow_register"

// registerEnabled 读取开放注册开关 (仅启用状态的配置生效, 查询失败/缺省视为关闭)。
// 演示环境恒为关: 注册会真实落库建账号, 公开演示站点会被批量注册灌脏数据。
func registerEnabled(ctx context.Context) bool {
	if demox.Enabled(ctx) {
		return false
	}
	v, err := dao.SysConfig.Ctx(ctx).
		Fields("config_value").
		Where("config_key", KeyAllowRegister).
		Where("status", consts.StatusEnabled).
		Where("deleted_at IS NULL").
		Value()
	if err != nil || v == nil {
		return false
	}
	s := v.String()
	return s == "true" || s == "1"
}

// RegisterStatus 查询注册开关 (公开接口, 登录页据此决定是否展示注册入口)。
func (s *sAuth) RegisterStatus(ctx context.Context, _ *v1.RegisterStatusReq) (res *v1.RegisterStatusRes, err error) {
	return &v1.RegisterStatusRes{AllowRegister: registerEnabled(ctx)}, nil
}

// Register 自助注册: 开关校验 -> RSA 解密 -> 密码策略 -> 建用户 -> 绑定内置普通角色。
// 注册的密码由用户本人设定, 不置强制改密标志 (区别于管理员代建账号)。
func (s *sAuth) Register(ctx context.Context, req *v1.RegisterReq) (res *v1.RegisterRes, err error) {
	if !registerEnabled(ctx) {
		return nil, xerror.New(xerror.CodeBusinessError, "注册功能未开放")
	}

	// 与登录同通道: 一次性公钥加密, 解密失败不落任何痕迹
	plainPassword, derr := decryptPassword(ctx, req.KeyId, req.Password)
	if derr != nil {
		return nil, derr
	}
	if perr := pwdpolicy.Validate(ctx, plainPassword); perr != nil {
		return nil, perr
	}

	// 用户名占用预检 (软删行不挡新注册); 软删残留同名行触发唯一键冲突时同样按已占用返回
	cnt, _ := dao.SysUser.Ctx(ctx).
		Where("username", req.Username).
		Where("deleted_at IS NULL").
		Count()
	if cnt > 0 {
		return nil, xerror.New(xerror.CodeUsernameExists)
	}

	nickname := req.Nickname
	if nickname == "" {
		nickname = req.Username
	}
	hash, herr := password.Hash(plainPassword)
	if herr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, herr, "密码加密失败")
	}
	id, ierr := dao.SysUser.Ctx(ctx).Data(g.Map{
		"username":        req.Username,
		"password":        hash,
		"nickname":        nickname,
		"org_id":          0,
		"status":          consts.StatusEnabled,
		"pwd_updated_at":  gtime.Now(),
		"must_change_pwd": 0,
	}).InsertAndGetId()
	if ierr != nil {
		// 常见为软删残留同名行触发 uk_username 唯一键冲突, 统一按"已存在"提示;
		// 其余 DB 错误也记录原始错误便于排查 (公开接口不向客户端透出底层细节)
		g.Log().Warningf(ctx, "Register insert user failed, username=%s err=%v", req.Username, ierr)
		return nil, xerror.New(xerror.CodeUsernameExists)
	}

	// 绑定内置普通角色 (固定ID=2): 角色被删/被禁用时跳过绑定仅告警,
	// 注册本身不因此失败 —— 无角色用户仍可登录, 由管理员后续补配。
	roleOk, _ := dao.SysRole.Ctx(ctx).
		Where("id", consts.RoleCommonId).
		Where("status", consts.StatusEnabled).
		Where("deleted_at IS NULL").
		Count()
	if roleOk > 0 {
		if serr := casbinx.SetUserRoles(ctx, uint64(id), []uint64{consts.RoleCommonId}); serr != nil {
			g.Log().Errorf(ctx, "Register SetUserRoles failed, userId=%d err=%v", id, serr)
			rollbackRegisterUser(ctx, id)
			return nil, xerror.Wrap(xerror.CodeBusinessError, serr, "分配默认角色失败")
		}
	} else {
		g.Log().Warningf(ctx, "Register default role missing, userId=%d created without role", id)
	}
	return &v1.RegisterRes{}, nil
}

// rollbackRegisterUser 物理删除刚插入的用户, 角色绑定失败时的补偿。
func rollbackRegisterUser(ctx context.Context, id int64) {
	if _, err := dao.SysUser.Ctx(ctx).Where("id", id).Delete(); err != nil {
		g.Log().Errorf(ctx, "rollbackRegisterUser failed, id=%d err=%v", id, err)
	}
}

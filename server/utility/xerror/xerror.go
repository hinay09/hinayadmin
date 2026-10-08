// Package xerror 统一业务错误码与构造器。
package xerror

import (
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

// 业务错误码: 50000+ 段位避开 GoFrame 默认码。
var (
	CodeUnauthorized = gcode.New(40100, "未登录或登录已过期", nil)
	CodeForbidden    = gcode.New(40300, "无访问权限", nil)
	// CodePwdMustChange 密码被管理员重置/导入或已过期, 需先完成改密。
	// 走 HTTP 428 Precondition Required (业务码 42800), 与 40300/HTTP 403 无权限
	// 彻底隔离, 避免前端或中间层任何通用 403 处理误伤, 前端据此跳转改密页。
	CodePwdMustChange  = gcode.New(42800, "密码已过期或被重置, 请先修改密码", nil)
	CodeNotFound       = gcode.New(40400, "资源不存在", nil)
	CodeTooManyReq     = gcode.New(42900, "请求过于频繁, 请稍后再试", nil)
	CodeParamInvalid   = gcode.New(40000, "请求参数不合法", nil)
	CodeBusinessError  = gcode.New(50000, "业务错误", nil)
	CodeUserNotFound   = gcode.New(50001, "用户不存在", nil)
	CodePasswordWrong  = gcode.New(50002, "用户名或密码错误", nil)
	CodeUserDisabled   = gcode.New(50003, "用户已被禁用", nil)
	CodeUsernameExists = gcode.New(50004, "用户名已存在", nil)
	CodeRoleCodeExists = gcode.New(50010, "角色编码已存在", nil)
	// CodeRsaKeyInvalid 登录加密密钥缺失/过期/解密失败, 前端应重新获取公钥后重试。
	CodeRsaKeyInvalid = gcode.New(50005, "登录加密已失效, 请重试", nil)
	// CodeTotpInvalid 两步验证动态码错误或已使用(重放)。
	CodeTotpInvalid = gcode.New(50020, "动态验证码错误或已失效", nil)
	// CodeTotpTicketInvalid 两步验证登录票据缺失/过期/失败次数超限, 前端应回到密码登录重试。
	CodeTotpTicketInvalid = gcode.New(50021, "两步验证会话已过期, 请重新登录", nil)
	// CodeCaptchaInvalid 图形验证码缺失/错误/已过期(一次性消费后即失效), 前端应刷新验证码后重试。
	CodeCaptchaInvalid = gcode.New(50022, "验证码错误或已失效", nil)
)

// New 构造业务错误。
func New(code gcode.Code, msg ...string) error {
	if len(msg) > 0 && msg[0] != "" {
		return gerror.NewCode(code, msg[0])
	}
	return gerror.NewCode(code, code.Message())
}

// Wrap 包装一个底层错误为业务错误。
func Wrap(code gcode.Code, err error, msg ...string) error {
	if err == nil {
		return nil
	}
	if len(msg) > 0 && msg[0] != "" {
		return gerror.WrapCode(code, err, msg[0])
	}
	return gerror.WrapCode(code, err, code.Message())
}

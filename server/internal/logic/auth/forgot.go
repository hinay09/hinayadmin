// Package auth 找回密码 (短信验证码): 公开两步流程。
//
//	POST /auth/password/forgot/send-code  账号(用户名/手机号) → 短信验证码
//	POST /auth/password/forgot/reset      账号 + 验证码 + 新密码(RSA 密文) → 重置
//
// 安全设计:
//   - 防爆破: 验证码 6 位/5 分钟/单次消费 (GETDEL), 单账号失败 5 次锁定 15 分钟;
//   - 防刷量: 单 IP 每分钟封顶 + 单账号发码间隔 60s + 单账号每日 10 条;
//   - 防脚本: 发码入口校验图形验证码 (sys.captcha_enable 开启时, 与登录/注册同款);
//   - 防枚举: 账号不存在/未绑定手机号时返回通用成功 (不发码); 响应恒不回显手机号
//     (正常发送与异常账号的响应完全一致);
//   - 新密码走登录同款一次性 RSA 公钥加密 + 密码策略校验; demo 模式下重置被拦截 (428)。
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/logic/pwdpolicy"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/internal/sms"
	"hinay.cn/admin/utility/demox"
	"hinay.cn/admin/utility/password"
	"hinay.cn/admin/utility/xerror"
)

// ForgotSendCode 发送找回密码短信验证码。
// 防护分层: 单 IP 限流 (兜底, 防换账号轰炸) → 图形验证码 (与登录/注册同款, 挡脚本)
// → 租户解析 → 单账号频控; 响应恒回空 maskedPhone, 不区分账号状态 (防枚举)。
func (s *sAuth) ForgotSendCode(ctx context.Context, req *v1.ForgotSendCodeReq) (res *v1.ForgotSendCodeRes, err error) {
	account := strings.TrimSpace(req.Account)
	ip, ua := clientIp(ctx), requestUserAgent(ctx)

	// 单 IP 限流: 按账号的间隔/日限换账号即可绕过, IP 维度封顶短信成本与轰炸面
	limitKey := consts.PwdResetSendIpLimitPrefix + ip
	if n, lerr := g.Redis().GroupString().Incr(ctx, limitKey); lerr == nil {
		if n == 1 {
			_, _ = g.Redis().GroupGeneric().Expire(ctx, limitKey, 60)
		} else if n > consts.PwdResetSendIpMaxPerMin {
			return nil, xerror.New(xerror.CodeTooManyReq, "发送过于频繁, 请稍后再试")
		}
	}

	// 图形验证码 (sys.captcha_enable 开启时): 置于租户解析之前 —— 与登录同款分层,
	// 机器人请求在昂贵/可探测操作前即被拦下 (验证码答案一次性消费)
	if captchaEnabled(ctx) && !verifyCaptcha(req.CaptchaId, req.CaptchaCode) {
		return nil, xerror.New(xerror.CodeCaptchaInvalid)
	}

	// 租户识别 (与登录同口径): 请求体 > X-Tenant-Code 头 > 平台租户
	tenant, terr := ResolveTenantForLogin(ctx, req.TenantCode)
	if terr != nil {
		service.LoginLog().Record(ctx, model.LoginLogEntry{
			TenantId: consts.TenantPlatformId, Username: account,
			Status: consts.LoginLogStatusFail, Message: "找回密码发码入口拦截: " + terr.Error(),
			Ip: ip, UserAgent: ua,
		})
		return nil, terr
	}
	ctx = ScopedTenantCtx(ctx, tenant)

	u, uerr := findLoginUser(ctx, account)
	if uerr != nil {
		return nil, uerr // 一号多户等账号歧义 (与登录同款提示)
	}
	if u == nil || !mobilePattern.MatchString(u.Phone) {
		// 防枚举: 账号不存在 / 未绑定手机号 → 通用成功, 不发码
		return &v1.ForgotSendCodeRes{}, nil
	}

	// 频控: 间隔锁 (SET NX EX 原子) + 当日上限
	gapKey := consts.PwdResetGapPrefix + fmt.Sprint(tenant.Id) + ":" + account
	if v, serr := g.Redis().Do(ctx, "SET", gapKey, 1, "NX", "EX", consts.PwdResetGapSec); serr == nil && (v == nil || v.IsNil()) {
		return nil, xerror.New(xerror.CodeBusinessError, "发送过于频繁, 请稍后再试")
	}
	dailyKey := consts.PwdResetDailyPrefix + fmt.Sprint(tenant.Id) + ":" + account
	if n, serr := g.Redis().GroupString().Incr(ctx, dailyKey); serr == nil {
		if n == 1 {
			_, _ = g.Redis().GroupGeneric().Expire(ctx, dailyKey, 86400)
		} else if n > consts.PwdResetDailyMax {
			return nil, xerror.New(xerror.CodeBusinessError, "今日验证码发送次数已达上限")
		}
	}

	code := randomCode6()
	codeKey := consts.PwdResetCodePrefix + fmt.Sprint(tenant.Id) + ":" + fmt.Sprint(u.Id)
	if serr := g.Redis().GroupString().SetEX(ctx, codeKey, code, consts.PwdResetCodeTTLSec); serr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, serr, "保存验证码失败")
	}
	if serr := sms.Send(ctx, u.Phone, code); serr != nil {
		// 发送失败回收验证码, 避免"已存码但用户收不到"的悬空态
		_, _ = g.Redis().Do(ctx, "DEL", codeKey)
		g.Log().Warningf(ctx, "找回密码短信发送失败(user %d): %v", u.Id, serr)
		return nil, xerror.New(xerror.CodeBusinessError, "短信发送失败, 请稍后再试或联系管理员")
	}
	service.LoginLog().Record(ctx, model.LoginLogEntry{
		TenantId: tenant.Id, UserId: u.Id, Username: account,
		Status: consts.LoginLogStatusSuccess, Message: "找回密码验证码已发送",
		Ip: ip, UserAgent: ua,
	})
	// 防枚举: 不回显打码手机号 —— 正常发送与"账号不存在/未绑定手机"的响应完全一致
	return &v1.ForgotSendCodeRes{}, nil
}

// ForgotReset 校验验证码并重置密码。
func (s *sAuth) ForgotReset(ctx context.Context, req *v1.ForgotResetReq) (res *v1.ForgotResetRes, err error) {
	if gerr := demox.Guard(ctx); gerr != nil {
		return nil, gerr // demo 模式拦截密码重置 (与改密/管理员重置同口径)
	}
	account := strings.TrimSpace(req.Account)
	ip, ua := clientIp(ctx), requestUserAgent(ctx)

	tenant, terr := ResolveTenantForLogin(ctx, req.TenantCode)
	if terr != nil {
		return nil, terr
	}
	ctx = ScopedTenantCtx(ctx, tenant)

	fail := func(msg string) error {
		key := consts.PwdResetFailPrefix + fmt.Sprint(tenant.Id) + ":" + account
		n, _ := g.Redis().GroupString().Incr(ctx, key)
		if n == 1 {
			_, _ = g.Redis().GroupGeneric().Expire(ctx, key, consts.LoginFailWindowSec)
		}
		service.LoginLog().Record(ctx, model.LoginLogEntry{
			TenantId: tenant.Id, Username: account,
			Status: consts.LoginLogStatusFail, Message: "找回密码重置失败: " + msg,
			Ip: ip, UserAgent: ua,
		})
		return xerror.New(xerror.CodeBusinessError, msg)
	}

	u, uerr := findLoginUser(ctx, account)
	if uerr != nil {
		return nil, uerr
	}
	if u == nil {
		return nil, fail("验证码错误或已过期") // 不区分账号不存在, 防枚举
	}
	// 失败锁定检查 (窗口内超限)
	if n, rerr := g.Redis().GroupString().Get(ctx, consts.PwdResetFailPrefix+fmt.Sprint(tenant.Id)+":"+account); rerr == nil && n != nil && !n.IsNil() && n.Int64() >= consts.PwdResetFailMax {
		return nil, xerror.New(xerror.CodeBusinessError, "失败次数过多, 请 15 分钟后再试")
	}

	// 验证码校验: GETDEL 单次消费
	codeKey := consts.PwdResetCodePrefix + fmt.Sprint(tenant.Id) + ":" + fmt.Sprint(u.Id)
	stored, cerr := g.Redis().GroupString().GetDel(ctx, codeKey)
	if cerr != nil || stored == nil || stored.IsNil() || stored.String() != strings.TrimSpace(req.Code) {
		return nil, fail("验证码错误或已过期")
	}

	// 新密码: 登录同款一次性 RSA 解密 + 长度/策略校验
	plain, derr := decryptPassword(ctx, req.KeyId, req.Password)
	if derr != nil {
		return nil, derr
	}
	if l := len([]rune(plain)); l < 6 || l > 32 {
		return nil, fail("密码长度 6-32")
	}
	if perr := pwdpolicy.Validate(ctx, plain); perr != nil {
		return nil, fail(perr.Error())
	}
	hash, herr := password.Hash(plain)
	if herr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, herr, "密码加密失败")
	}
	if _, uerr := dao.SysUser.Ctx(ctx).Where("id", u.Id).
		Data(g.Map{
			"password":        hash,
			"pwd_updated_at":  gtime.Now(),
			"must_change_pwd": 0, // 用户主动重置, 视为已通过身份验证
		}).Update(); uerr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, uerr, "重置失败")
	}
	// 重置成功清除失败计数
	_, _ = g.Redis().Do(ctx, "DEL", consts.PwdResetFailPrefix+fmt.Sprint(tenant.Id)+":"+account)
	service.LoginLog().Record(ctx, model.LoginLogEntry{
		TenantId: tenant.Id, UserId: u.Id, Username: account,
		Status: consts.LoginLogStatusSuccess, Message: "找回密码重置成功",
		Ip: ip, UserAgent: ua,
	})
	return &v1.ForgotResetRes{}, nil
}

// randomCode6 6 位数字验证码 (密码学随机, 首位可为 0)。
func randomCode6() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		// crypto/rand 失败极罕见: 退化为 32 位随机 hex 截取仍不可预测
		b := make([]byte, 4)
		_, _ = rand.Read(b)
		s := hex.EncodeToString(b)
		return s[:6]
	}
	return fmt.Sprintf("%06d", n.Int64())
}

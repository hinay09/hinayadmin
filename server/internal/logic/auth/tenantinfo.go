// tenantinfo.go —— 登录入口的租户识别 (docs/multi-tenant.md §4.3) 与公开租户信息接口。
package auth

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/tenantx"
	"hinay.cn/admin/utility/xerror"
)

// ResolveTenantForLogin 登录/注册入口的租户解析 (D3: A+B 组合)。
// 优先级: 显式 code 参数 > X-Tenant-Code 请求头 (子域名方案由前端从 hostname 提取后塞头)
// > 默认平台租户。三态错误: 租户码无效 / 已禁用 / 已到期 各有专属错误码, 不静默失败。
// 关闭态恒返回平台租户 (行为与单租户一致)。
func ResolveTenantForLogin(ctx context.Context, code string) (*model.TenantLite, error) {
	if !tenantx.Enabled(ctx) {
		return &model.TenantLite{Id: consts.TenantPlatformId, Code: consts.TenantPlatformCode, Name: "平台租户", Status: consts.StatusEnabled}, nil
	}
	if code = strings.TrimSpace(code); code == "" {
		if r := ghttp.RequestFromCtx(ctx); r != nil {
			code = strings.TrimSpace(r.Header.Get(consts.HeaderTenantCode))
		}
	}
	if code == "" {
		code = consts.TenantPlatformCode
	}
	t, err := service.Tenant().LoadLiteByCode(ctx, code)
	if err != nil {
		return nil, xerror.New(xerror.CodeTenantCodeInvalid)
	}
	if t.Status != consts.StatusEnabled {
		return nil, xerror.New(xerror.CodeTenantDisabled)
	}
	if t.ExpireAt != "" {
		// 轻量行 ExpireAt 为 "Y-m-d H:i:s" 文本, 交给 Auth 中间件逐请求精确校验,
		// 此处仅作入口快速拦截 (parse 失败不拦截, 由中间层兜底)
		if ft, perr := gtime.StrToTime(t.ExpireAt); perr == nil && ft.Before(gtime.Now()) {
			return nil, xerror.New(xerror.CodeTenantExpired)
		}
	}
	return t, nil
}

// TenantInfo 公开租户信息 (仅显示名; 按 IP 限流防租户码枚举)。
func (s *sAuth) TenantInfo(ctx context.Context, req *v1.TenantInfoReq) (res *v1.TenantInfoRes, err error) {
	// 单 IP 每分钟限流 (复用 LoginFailPrefix 的 Redis 计数模式)
	limitKey := consts.TenantInfoLimitPrefix + clientIp(ctx)
	if n, lerr := g.Redis().GroupString().Incr(ctx, limitKey); lerr == nil {
		if n == 1 {
			_, _ = g.Redis().GroupGeneric().Expire(ctx, limitKey, 60)
		} else if n > consts.TenantInfoMaxPerMin {
			return nil, xerror.New(xerror.CodeTooManyReq, "查询过于频繁, 请稍后再试")
		}
	}
	out := &v1.TenantInfoRes{MultiTenant: tenantx.Enabled(ctx)}
	if !out.MultiTenant || strings.TrimSpace(req.Code) == "" {
		return out, nil
	}
	t, terr := service.Tenant().LoadLiteByCode(ctx, strings.TrimSpace(req.Code))
	if terr != nil {
		// 不区分"不存在/禁用"细节, 仅返回 found=false (防枚举: 显示名只在存在时给出)
		return out, nil
	}
	out.Name = t.Name
	out.Found = true
	out.Enabled = t.Status == consts.StatusEnabled
	return out, nil
}

// ScopedTenantCtx 构造带注入的租户上下文 (登录/注册公开接口在无中间件组装时使用)。
func ScopedTenantCtx(ctx context.Context, t *model.TenantLite) context.Context {
	return contextx.WithTenant(ctx, &model.TenantInfo{Id: t.Id, Code: t.Code, Name: t.Name, Inject: true})
}

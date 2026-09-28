// Package pwdpolicy 密码策略 (配置驱动, 全局配置 sys.password.* 键)。
//
// 复杂度校验作用于所有"设置新密码"的入口 (改密/创建/重置/导入);
// 登录时不校验旧密码 (不能因策略提高而把人挡在门外), 但会检查:
//   - must_change_pwd 标志 (管理员创建/重置/导入默认密码后强制改密);
//   - 密码有效期 (pwd_updated_at + expire_days, 0 = 永不过期)。
//
// 策略键 (sys_config, 管理后台"全局配置"页可直接修改):
//
//	sys.password.min_length       最小长度 (默认 6)
//	sys.password.max_length       最大长度 (默认 32)
//	sys.password.require_upper    需含大写字母
//	sys.password.require_lower    需含小写字母
//	sys.password.require_digit    需含数字
//	sys.password.require_special  需含特殊字符 (!@#$%^&* 等)
//	sys.password.expire_days      有效期天数 (0 = 永不过期)
package pwdpolicy

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gogf/gf/v2/os/gtime"

	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/utility/xerror"
)

// 策略配置键。
const (
	KeyMinLength      = "sys.password.min_length"
	KeyMaxLength      = "sys.password.max_length"
	KeyRequireUpper   = "sys.password.require_upper"
	KeyRequireLower   = "sys.password.require_lower"
	KeyRequireDigit   = "sys.password.require_digit"
	KeyRequireSpecial = "sys.password.require_special"
	KeyExpireDays     = "sys.password.expire_days"
)

// allKeys 策略全部键 (一次性查询用)。
var allKeys = []string{
	KeyMinLength, KeyMaxLength, KeyRequireUpper, KeyRequireLower,
	KeyRequireDigit, KeyRequireSpecial, KeyExpireDays,
}

// Policy 一次查询得到的策略快照。
type Policy struct {
	MinLength      int
	MaxLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireDigit   bool
	RequireSpecial bool
	ExpireDays     int
}

// defaults 与现有校验 (长度 6-32) 保持一致的缺省策略。
func defaults() Policy {
	return Policy{MinLength: 6, MaxLength: 32}
}

// 策略 30s 内存缓存: Auth 中间件每个业务请求都会做改密判定,
// 避免逐请求读配置表 (配置修改最迟 30s 生效)。
var (
	policyMu    sync.RWMutex
	policyCache Policy
	policyAt    time.Time
)

// policyCacheTTL 策略缓存时长。
const policyCacheTTL = 30 * time.Second

// Load 读取当前启用的密码策略 (30s 缓存; 查询失败/缺项回退默认值, 不阻断业务)。
func Load(ctx context.Context) Policy {
	policyMu.RLock()
	if !policyAt.IsZero() && time.Since(policyAt) < policyCacheTTL {
		p := policyCache
		policyMu.RUnlock()
		return p
	}
	policyMu.RUnlock()

	p := loadFromDB(ctx)

	policyMu.Lock()
	policyCache, policyAt = p, time.Now()
	policyMu.Unlock()
	return p
}

// loadFromDB 实际查询配置表。
func loadFromDB(ctx context.Context) Policy {
	p := defaults()
	rows, err := dao.SysConfig.Ctx(ctx).
		Fields("config_key, config_value").
		WhereIn("config_key", allKeys).
		Where("status", 1).
		Where("deleted_at IS NULL").
		All()
	if err != nil {
		return p
	}
	get := func(key string) (string, bool) {
		for _, r := range rows {
			if r["config_key"].String() == key {
				return r["config_value"].String(), true
			}
		}
		return "", false
	}
	if v, ok := get(KeyMinLength); ok && v != "" {
		if n := parsePositive(v); n > 0 {
			p.MinLength = n
		}
	}
	if v, ok := get(KeyMaxLength); ok && v != "" {
		if n := parsePositive(v); n > 0 {
			p.MaxLength = n
		}
	}
	if p.MaxLength < p.MinLength {
		p.MaxLength = p.MinLength
	}
	v, _ := get(KeyRequireUpper)
	p.RequireUpper = parseBool(v)
	v, _ = get(KeyRequireLower)
	p.RequireLower = parseBool(v)
	v, _ = get(KeyRequireDigit)
	p.RequireDigit = parseBool(v)
	v, _ = get(KeyRequireSpecial)
	p.RequireSpecial = parseBool(v)
	if v, ok := get(KeyExpireDays); ok && v != "" {
		p.ExpireDays = parsePositive(v)
	}
	return p
}

// Validate 按当前策略校验新密码明文, 不满足返回带明确原因的业务错误。
func Validate(ctx context.Context, plain string) error {
	p := Load(ctx)
	r := []rune(plain)
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range r {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		case !unicode.IsLetter(c) && !unicode.IsDigit(c):
			hasSpecial = true
		}
	}
	if len(r) < p.MinLength || len(r) > p.MaxLength {
		return xerror.New(xerror.CodeParamInvalid,
			fmt.Sprintf("密码长度需 %d-%d 位", p.MinLength, p.MaxLength))
	}
	var missing []string
	if p.RequireUpper && !hasUpper {
		missing = append(missing, "大写字母")
	}
	if p.RequireLower && !hasLower {
		missing = append(missing, "小写字母")
	}
	if p.RequireDigit && !hasDigit {
		missing = append(missing, "数字")
	}
	if p.RequireSpecial && !hasSpecial {
		missing = append(missing, "特殊字符")
	}
	if len(missing) > 0 {
		return xerror.New(xerror.CodeParamInvalid,
			"密码需包含："+strings.Join(missing, "、"))
	}
	return nil
}

// Expired 密码是否已过期 (expire_days=0 永不过期; 从未改过密码视为已过期,
// 促使存量账号在启用有效期后完成一次改密)。
func Expired(ctx context.Context, pwdUpdatedAt *gtime.Time) bool {
	p := Load(ctx)
	if p.ExpireDays <= 0 {
		return false
	}
	if pwdUpdatedAt == nil {
		return true
	}
	return gtime.Now().After(pwdUpdatedAt.AddDate(0, 0, p.ExpireDays))
}

// parsePositive 正整数解析, 非法回退 0。
func parsePositive(s string) int {
	n := 0
	for _, c := range strings.TrimSpace(s) {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
		if n > 1<<30 {
			return 0
		}
	}
	return n
}

// parseBool 宽松布尔解析 (与前端配置存储的 "true"/"false" 文本对齐)。
func parseBool(s string) bool {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "true", "1", "yes", "on":
		return true
	}
	return false
}

// Package consts 全局常量定义。
package consts

// ctxKey 是未导出的自定义类型, 避免跨包起 context key 碰撞 (遵循 Go 标准 context 包最佳实践)。
type ctxKey int

// Context Key (类型安全, 只能通过 consts.* 常量读写)
const (
	// CtxUserKey 上下文中存储登录用户信息的 key。
	CtxUserKey ctxKey = iota + 1
	// CtxRequestIdKey 请求 ID。
	CtxRequestIdKey
	// CtxJwtTokenKey 请求 JWT 原始 token。
	CtxJwtTokenKey
)

// CtxUserKeyName 供兼容层使用的字符串 key名 (仅用于 ghttp.Request.SetCtxVar/GetCtxVar 这种需要可迭代 string 的场景)。
const (
	CtxUserKeyName     = "ctxLoginUser"
	CtxRequestIdName   = "RequestId"
	CtxJwtTokenKeyName = "jwtToken"
)

// 鉴权相关
const (
	// AuthHeader 鉴权请求头。
	AuthHeader = "Authorization"
	// AuthScheme Bearer 前缀。
	AuthScheme = "Bearer "
	// JWTBlacklistPrefix Redis 中 JWT 黑名单 Key 前缀。
	JWTBlacklistPrefix = "hinay:jwt:black:"
	// LoginFailPrefix Redis 中登录失败计数 Key 前缀。
	LoginFailPrefix = "hinay:login:fail:"
)

// 登录防暴力破解配置。
const (
	// LoginFailMax 统计窗口内允许的最大失败次数, 超过后锁定。
	LoginFailMax = 5
	// LoginFailWindowSec 失败计数统计窗口(秒)。
	LoginFailWindowSec = 900
)

// 通用状态
const (
	StatusEnabled  = 1 // 启用
	StatusDisabled = 0 // 禁用
)

// 菜单类型
const (
	MenuTypeDir    = 1 // 目录
	MenuTypeMenu   = 2 // 菜单
	MenuTypeButton = 3 // 按钮
)

// 内置角色
const (
	RoleAdmin = "admin"
)

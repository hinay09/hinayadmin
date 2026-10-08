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
	// CtxAuditUploadName 上传类请求在 ctx 中暂存的文件摘要,
	// OperationLog defer 阶段读取后作为审计 detail (multipart 不读请求体)。
	CtxAuditUploadName = "ctxAuditUpload"
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
	// RSAKeyPrefix Redis 中一次性 RSA 私钥 Key 前缀。
	RSAKeyPrefix = "hinay:rsa:priv:"
	// PubKeyLimitPrefix 公钥接口限流计数 Key 前缀。
	PubKeyLimitPrefix = "hinay:rsa:limit:"
)

// 在线会话 (Redis)
const (
	// OnlineSessionKey 在线会话 HASH Key, field=会话ID(token摘要), value=会话详情 JSON。
	OnlineSessionKey = "hinay:online:sessions"
	// OnlineTouchPrefix 心跳节流 Key 前缀, 命中 NX 才真正刷新会话活跃时间。
	OnlineTouchPrefix = "hinay:online:touch:"
	// OnlineTouchIntervalSec 会话活跃时间刷新节流间隔(秒)。
	OnlineTouchIntervalSec = 60
)

// 登录防暴力破解配置。
const (
	// LoginFailMax 统计窗口内允许的最大失败次数, 超过后锁定。
	LoginFailMax = 5
	// LoginFailWindowSec 失败计数统计窗口(秒)。
	LoginFailWindowSec = 900
)

// 登录密码 RSA 加密配置。
const (
	// RSAKeyTTLSec 一次性私钥存活时间(秒), 覆盖取公钥到提交登录的时间窗。
	RSAKeyTTLSec = 120
	// PubKeyMaxPerMin 公钥接口单 IP 每分钟最大请求次数(防密钥生成 DoS)。
	PubKeyMaxPerMin = 20
)

// 图形验证码 (登录/注册公开入口的人机挑战, 开源库 base64Captcha) 配置。
const (
	// CaptchaPrefix Redis 中验证码答案 Key 前缀 (值为答案, GETDEL 一次性消费)。
	CaptchaPrefix = "hinay:captcha:"
	// CaptchaLimitPrefix 验证码图片接口单 IP 限流计数 Key 前缀。
	CaptchaLimitPrefix = "hinay:captcha:limit:"
	// CaptchaTTLSec 验证码存活时间(秒): 获取图片到提交登录/注册的时间窗。
	CaptchaTTLSec = 300
	// CaptchaMaxPerMin 验证码图片接口单 IP 每分钟最大请求次数(防图片渲染 DoS)。
	CaptchaMaxPerMin = 30
)

// TOTP 两步验证配置。
const (
	// TotpTicketPrefix Redis 中两步验证登录票据 Key 前缀 (值=用户ID)。
	TotpTicketPrefix = "hinay:totp:ticket:"
	// TotpTicketFailPrefix Redis 中两步验证票据失败计数 Key 前缀。
	TotpTicketFailPrefix = "hinay:totp:fail:"
	// TotpTicketTTLSec 票据存活时间(秒): 密码通过到提交动态码的时间窗。
	TotpTicketTTLSec = 300
	// TotpTicketFailMax 单票据允许的动态码最大失败次数, 超出后票据作废需重新走密码登录。
	TotpTicketFailMax = 5
)

// 通用状态
const (
	StatusEnabled  = 1 // 启用
	StatusDisabled = 0 // 禁用
)

// 登录日志结果
const (
	LoginLogStatusSuccess = 1 // 登录成功
	LoginLogStatusFail    = 0 // 登录失败
)

// 任务执行日志结果
const (
	JobLogStatusSuccess = 1 // 执行成功
	JobLogStatusFail    = 0 // 执行失败
)

// 数据范围 (sys_role.data_scope, 组织架构数据权限)
const (
	DataScopeAll     = 1 // 全部数据
	DataScopeCustom  = 2 // 自定义组织 (sys_role_org)
	DataScopeDept    = 3 // 本部门
	DataScopeDeptSub = 4 // 本部门及以下
	DataScopeSelf    = 5 // 仅本人
)

// 菜单类型
const (
	MenuTypeDir    = 1 // 目录
	MenuTypeMenu   = 2 // 菜单
	MenuTypeButton = 3 // 按钮
)

// 内置角色ID: 超级管理员角色固定为 1 (种子数据/删除保护/权限配置豁免均按此ID判定,
// 角色code仅为展示标识, 不参与权限匹配); 普通用户角色固定为 2, 自助注册默认绑定。
const (
	RoleAdminId  uint64 = 1
	RoleCommonId uint64 = 2
)

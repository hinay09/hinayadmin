// Package twofactor TOTP 两步验证 (RFC 6238)。
//
// 绑定流程: setup 生成密钥(enabled=0) -> 用户扫码/手输 -> enable 校验动态码生效(enabled=1);
// 登录流程: 密码通过后签发一次性票据, TotpLogin 携带票据 + 动态码换 token (见 logic/auth)。
//
// 防重放: 每次校验命中后把 last_step 推进到命中的时间步 (Unix 秒/30),
// 后续校验要求 step 严格大于 last_step, 同一动态码因此不可二次使用。
package twofactor

import (
	"context"
	"crypto/subtle"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	v1 "hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model/entity"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/demox"
	"hinay.cn/admin/utility/xerror"
)

// TOTP 参数: 与 Google Authenticator / Microsoft Authenticator 等主流验证器默认兼容
// (RFC 6238 推荐 SHA-1 + 6 位 + 30s 周期, 改动会导致用户端无法适配)。
const (
	totpPeriod uint = 30 // 时间步长(秒)
	totpSkew   int  = 1  // 允许的时钟偏移窗口数 (±30s)
)

// codePattern 动态码格式: 恰好 6 位数字 (容许用户复制带空格)。
var codePattern = regexp.MustCompile(`^[0-9]{6}$`)

type sTwoFactor struct{}

func init() {
	service.RegisterTwoFactor(NewTwoFactor())
}

func NewTwoFactor() *sTwoFactor {
	return &sTwoFactor{}
}

// normalizeCode 去空白并校验 6 位数字格式, 不合法直接报参数错误。
func normalizeCode(code string) (string, error) {
	c := strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if !codePattern.MatchString(c) {
		return "", xerror.New(xerror.CodeParamInvalid, "动态验证码为 6 位数字")
	}
	return c, nil
}

// issuer otpauth URI 中的发行方名称, 取全局配置 sys.name (验证器 App 里展示的站点名)。
func issuer(ctx context.Context) string {
	if v, err := dao.SysConfig.Ctx(ctx).
		Where("config_key", "sys.name").
		Where("status", 1).
		Where("deleted_at IS NULL").
		Value("config_value"); err == nil && v != nil {
		if s := strings.TrimSpace(v.String()); s != "" {
			return s
		}
	}
	return "hinay-admin"
}

// loadRow 取用户当前绑定记录 (无记录返回 nil)。
func loadRow(ctx context.Context, userId uint64) (*entity.SysUserTotp, error) {
	var row *entity.SysUserTotp
	err := dao.SysUserTotp.Ctx(ctx).
		Where("user_id", userId).
		Where("deleted_at IS NULL").
		Scan(&row)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "查询两步验证配置失败")
	}
	return row, nil
}

// matchCode 在 [now-skew, now+skew] 窗口内逐时间步生成期望码并常量时间比较。
// 返回命中的时间步; step <= lastStep 视为重放, 拒绝。
func matchCode(secret, code string, now time.Time, lastStep int64) (int64, bool) {
	opts := totp.ValidateOpts{
		Period:    totpPeriod,
		Skew:      0, // 窗口展开在此处自行遍历, 便于拿到命中步
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	}
	for offset := -totpSkew; offset <= totpSkew; offset++ {
		t := now.Add(time.Duration(offset*int(totpPeriod)) * time.Second)
		want, err := totp.GenerateCodeCustom(secret, t, opts)
		if err != nil {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) != 1 {
			continue
		}
		step := t.Unix() / int64(totpPeriod)
		if step <= lastStep {
			// 已消费过的验证码 (或更早窗口), 重放拒绝
			return 0, false
		}
		return step, true
	}
	return 0, false
}

// advanceStep 单调推进 last_step: 条件更新保证并发下只进不退。
func advanceStep(ctx context.Context, userId uint64, step int64) bool {
	res, err := dao.SysUserTotp.Ctx(ctx).
		Where("user_id", userId).
		Where("last_step < ?", step).
		Data(g.Map{"last_step": step}).
		Update()
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

// Enabled 用户是否已开启两步验证。
//
// 可选性契约 (fail-open): 两步验证是用户自选项, 本查询的任何失败 (含部署后未执行
// 0011_totp.sql 建表) 一律按"未开启"降级, 绝不阻断未 opt-in 用户的登录与个人中心;
// 仅告警一次提示补跑升级脚本。仅当绑定记录存在且 enabled=1 时才返回 true,
// 即: 未绑定用户的所有既有流程与无本功能时逐字节一致。
func (s *sTwoFactor) Enabled(ctx context.Context, userId uint64) bool {
	row, err := loadRow(ctx, userId)
	if err != nil {
		warnQueryOnce.Do(func() {
			g.Log().Warningf(ctx,
				"查询 sys_user_totp 失败, 两步验证按未开启降级 (登录不受影响; 若未建表请执行 manifest/sql/init.sql): %v", err)
		})
		return false
	}
	if row == nil {
		return false
	}
	return row.Enabled == 1
}

// warnQueryOnce Enabled 查询失败的进程级一次性告警, 避免每次登录/userInfo 刷屏。
var warnQueryOnce sync.Once

// Setup 生成/重置当前用户的绑定密钥 (待验证状态)。
// 重复调用会覆盖旧密钥; 已启用的用户须先解绑再重新绑定。
// 演示环境禁止: 共享演示账号一旦绑定, 其他演示用户将无法通过两步验证登录。
func (s *sTwoFactor) Setup(ctx context.Context, req *v1.TotpSetupReq) (res *v1.TotpSetupRes, err error) {
	if gerr := demox.Guard(ctx); gerr != nil {
		return nil, gerr
	}
	cur := contextx.LoginUser(ctx)
	if cur == nil {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	if row, lerr := loadRow(ctx, cur.UserId); lerr != nil {
		return nil, lerr
	} else if row != nil && row.Enabled == 1 {
		return nil, xerror.New(xerror.CodeBusinessError, "两步验证已开启, 如需更换请先解绑")
	}

	key, gerr := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer(ctx),
		AccountName: cur.Username,
		Period:      totpPeriod,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if gerr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, gerr, "生成两步验证密钥失败")
	}

	// upsert: 首次绑定插入, 未完成的旧密钥直接覆盖
	_, err = dao.SysUserTotp.Ctx(ctx).
		Data(g.Map{
			"user_id": cur.UserId,
			"secret":  key.Secret(),
			"enabled": 0,
		}).
		Save()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "保存两步验证密钥失败")
	}
	return &v1.TotpSetupRes{Secret: key.Secret(), Otpauth: key.URL()}, nil
}

// Enable 确认绑定: 校验动态码通过后正式生效, 同时消费该码 (计入 last_step)。
func (s *sTwoFactor) Enable(ctx context.Context, req *v1.TotpEnableReq) (res *v1.TotpEnableRes, err error) {
	if gerr := demox.Guard(ctx); gerr != nil {
		return nil, gerr
	}
	cur := contextx.LoginUser(ctx)
	if cur == nil {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	c, cerr := normalizeCode(req.Code)
	if cerr != nil {
		return nil, cerr
	}
	row, err := loadRow(ctx, cur.UserId)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, xerror.New(xerror.CodeBusinessError, "请先生成两步验证密钥")
	}
	if row.Enabled == 1 {
		return nil, xerror.New(xerror.CodeBusinessError, "两步验证已开启")
	}
	step, ok := matchCode(row.Secret, c, time.Now(), row.LastStep)
	if !ok {
		return nil, xerror.New(xerror.CodeTotpInvalid)
	}
	// 推进 last_step 再置启用, 保证用于绑定的验证码不能在登录时复用
	if _, uerr := dao.SysUserTotp.Ctx(ctx).
		Where("user_id", cur.UserId).
		Data(g.Map{"enabled": 1, "last_step": step}).
		Update(); uerr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, uerr, "绑定两步验证失败")
	}
	return &v1.TotpEnableRes{}, nil
}

// Disable 解绑: 须提供当前有效动态码; 物理删除记录 (uk_user_id 与软删残留行冲突, 故不走软删)。
func (s *sTwoFactor) Disable(ctx context.Context, req *v1.TotpDisableReq) (res *v1.TotpDisableRes, err error) {
	if gerr := demox.Guard(ctx); gerr != nil {
		return nil, gerr
	}
	cur := contextx.LoginUser(ctx)
	if cur == nil {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	c, cerr := normalizeCode(req.Code)
	if cerr != nil {
		return nil, cerr
	}
	row, err := loadRow(ctx, cur.UserId)
	if err != nil {
		return nil, err
	}
	if row == nil || row.Enabled != 1 {
		return nil, xerror.New(xerror.CodeBusinessError, "两步验证未开启")
	}
	if _, ok := matchCode(row.Secret, c, time.Now(), row.LastStep); !ok {
		return nil, xerror.New(xerror.CodeTotpInvalid)
	}
	if _, derr := dao.SysUserTotp.Ctx(ctx).Unscoped().
		Where("user_id", cur.UserId).
		Delete(); derr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, derr, "解绑两步验证失败")
	}
	return &v1.TotpDisableRes{}, nil
}

// VerifyLogin 登录第二步动态码校验, 命中后推进 last_step 防重放。
func (s *sTwoFactor) VerifyLogin(ctx context.Context, userId uint64, code string) error {
	c, cerr := normalizeCode(code)
	if cerr != nil {
		return cerr
	}
	row, err := loadRow(ctx, userId)
	if err != nil {
		return err
	}
	if row == nil || row.Enabled != 1 {
		// 绑定记录在两步之间被解绑: 按验证码错误处理, 不泄露绑定状态
		return xerror.New(xerror.CodeTotpInvalid)
	}
	step, ok := matchCode(row.Secret, c, time.Now(), row.LastStep)
	if !ok {
		return xerror.New(xerror.CodeTotpInvalid)
	}
	if !advanceStep(ctx, userId, step) {
		// 并发重放: 同一码已在另一请求中被消费
		return xerror.New(xerror.CodeTotpInvalid)
	}
	return nil
}

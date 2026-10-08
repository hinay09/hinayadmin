// 图形验证码: 登录/注册公开入口的人机挑战 (开源库 github.com/mojocn/base64Captcha)。
// 开关: 全局配置 sys.captcha_enable (与 sys.allow_register 同机制, 管理员可在
// 全局配置页随时切换, 无需重启); 答案存 Redis 一次性消费 (GETDEL 原子取删),
// 与一次性 RSA 私钥/TOTP 票据同一套模式, 多副本部署天然共享, 重启不丢。
package auth

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/mojocn/base64Captcha"

	v1 "hinay.cn/admin/api/auth/v1"
	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/utility/xerror"
)

// KeyCaptchaEnable 验证码开关键 (与 init.sql sys_config 种子对齐)。
const KeyCaptchaEnable = "sys.captcha_enable"

// captchaEnabled 读取验证码开关 (仅启用状态的配置生效)。
// 查询失败/配置缺失按"开启"处理: 公开登录入口的挑战机制默认应该是有的,
// 配置系统故障不应静默降级为无挑战 (fail-close)。
func captchaEnabled(ctx context.Context) bool {
	v, err := dao.SysConfig.Ctx(ctx).
		Fields("config_value").
		Where("config_key", KeyCaptchaEnable).
		Where("status", consts.StatusEnabled).
		Where("deleted_at IS NULL").
		Value()
	if err != nil || v == nil {
		return true
	}
	s := v.String()
	return s != "false" && s != "0"
}

// redisCaptchaStore base64Captcha.Store 的 Redis 实现 (替代默认内存 Store:
// 内存 Store 单副本隔离且重启即失效)。Store 接口不携带 ctx, 此处均为
// 短命令, 用 background context 即可 (与 OperationLog 异步落库同模式)。
type redisCaptchaStore struct{}

// Set 写入验证码答案, TTL 内有效。
func (s *redisCaptchaStore) Set(id, value string) error {
	return g.Redis().GroupString().SetEX(context.Background(), consts.CaptchaPrefix+id, value, consts.CaptchaTTLSec)
}

// Get 读取答案; clear=true 时 GETDEL 原子消费 (用过即废, 不可重放)。
func (s *redisCaptchaStore) Get(id string, clear bool) string {
	ctx := context.Background()
	key := consts.CaptchaPrefix + id
	if clear {
		v, err := g.Redis().GroupString().GetDel(ctx, key)
		if err != nil || v == nil || v.IsNil() {
			return ""
		}
		return v.String()
	}
	v, err := g.Redis().GroupString().Get(ctx, key)
	if err != nil || v == nil || v.IsNil() {
		return ""
	}
	return v.String()
}

// Verify 校验答案 (大小写不敏感, 忽略首尾空白)。
func (s *redisCaptchaStore) Verify(id, answer string, clear bool) bool {
	return strings.EqualFold(strings.TrimSpace(s.Get(id, clear)), strings.TrimSpace(answer))
}

// captchaDriver 数学算式验证码 (如 "3+5=?"): 无大小写/形近字母困扰, 对人更友好,
// 答案是数字组合, 配合空心线+细直线干扰防 OCR。尺寸与登录页输入框行高对齐。
var captchaDriver = base64Captcha.NewDriverMath(
	40, 128, 4,
	base64Captcha.OptionShowHollowLine|base64Captcha.OptionShowSlimeLine,
	nil, nil, nil,
)

// Captcha 生成图形验证码 (公开接口)。
// 按 IP 限流防止匿名端点被刷导致图片渲染 DoS (与公钥接口同模式)。
func (s *sAuth) Captcha(ctx context.Context, _ *v1.CaptchaReq) (res *v1.CaptchaRes, err error) {
	if !captchaEnabled(ctx) {
		return &v1.CaptchaRes{CaptchaEnabled: false}, nil
	}

	// 单 IP 每分钟限流
	limitKey := consts.CaptchaLimitPrefix + clientIp(ctx)
	if n, lerr := g.Redis().GroupString().Incr(ctx, limitKey); lerr == nil {
		if n == 1 {
			_, _ = g.Redis().GroupGeneric().Expire(ctx, limitKey, 60)
		} else if n > consts.CaptchaMaxPerMin {
			return nil, xerror.New(xerror.CodeTooManyReq)
		}
	}

	c := base64Captcha.NewCaptcha(captchaDriver, &redisCaptchaStore{})
	id, b64s, _, gerr := c.Generate()
	if gerr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, gerr, "生成验证码失败")
	}
	return &v1.CaptchaRes{CaptchaEnabled: true, CaptchaId: id, Image: b64s}, nil
}

// verifyCaptcha 校验并一次性消费验证码。
// 空 id/空答案直接判败 (区分于"开关关闭时不校验", 由调用方控制)。
func verifyCaptcha(captchaId, code string) bool {
	if captchaId == "" || code == "" {
		return false
	}
	return base64Captcha.NewCaptcha(captchaDriver, &redisCaptchaStore{}).
		Verify(captchaId, code, true)
}

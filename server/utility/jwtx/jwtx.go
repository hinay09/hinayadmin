// Package jwtx JWT 签发与解析。
package jwtx

import (
	"context"
	"strings"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/golang-jwt/jwt/v5"
)

// Claims 自定义 JWT 载荷。
type Claims struct {
	UserId   uint64 `json:"uid"`
	Username string `json:"usn"`
	jwt.RegisteredClaims
}
type config struct {
	Secret    string
	ExpireSec int64
	Issuer    string
}

// minSecretLen HS256 对称签名密钥的最小长度(256bit)。
const minSecretLen = 32

// knownWeakSecrets 公开仓库中可见的历史默认密钥, 使用它们等于允许任何人伪造 token。
var knownWeakSecrets = map[string]struct{}{
	"hinay-admin-secret":            {},
	"hinay-admin-please-change-me":  {},
	"CHANGE-ME-openssl-rand-hex-32": {},
}

// loadConfig 加载并校验 JWT 配置。
// 安全约束: secret 缺失/过短/为公开默认值时直接报错(fail-close), 不允许带弱密钥运行。
func loadConfig(ctx context.Context) (*config, error) {
	cfg := g.Cfg()
	v, err := cfg.Get(ctx, "jwt.secret")
	if err != nil {
		return nil, gerror.Wrap(err, "读取 jwt.secret 配置失败")
	}
	secret := strings.TrimSpace(v.String())
	if secret == "" {
		return nil, gerror.New(`jwt.secret 未配置: 请在配置文件中设置随机密钥(可用 "openssl rand -hex 32" 生成)`)
	}
	if len(secret) < minSecretLen {
		return nil, gerror.Newf("jwt.secret 长度不足: 当前 %d 字符, 至少需要 %d 位随机字符", len(secret), minSecretLen)
	}
	if _, weak := knownWeakSecrets[secret]; weak {
		return nil, gerror.New(`jwt.secret 仍为公开默认值, 任何人可伪造管理员 token: 请更换为随机密钥(可用 "openssl rand -hex 32" 生成)`)
	}
	expire, _ := cfg.Get(ctx, "jwt.expireSec", 86400)
	issuer, _ := cfg.Get(ctx, "jwt.issuer", "hinay-admin")
	return &config{
		Secret:    secret,
		ExpireSec: expire.Int64(),
		Issuer:    issuer.String(),
	}, nil
}

// Generate 签发 token, 返回 token 字符串与到期时间戳(秒)。
func Generate(ctx context.Context, userId uint64, username string) (string, int64, error) {
	c, err := loadConfig(ctx)
	if err != nil {
		return "", 0, err
	}
	now := time.Now()
	exp := now.Add(time.Duration(c.ExpireSec) * time.Second)
	claims := &Claims{
		UserId:   userId,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    c.Issuer,
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := t.SignedString([]byte(c.Secret))
	if err != nil {
		return "", 0, err
	}
	return str, exp.Unix(), nil
}

// Parse 解析 token, 返回 Claims。
func Parse(ctx context.Context, tokenStr string) (*Claims, error) {
	c, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	claims := &Claims{}
	tk, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, gerror.New("unexpected signing method")
		}
		return []byte(c.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !tk.Valid {
		return nil, gerror.New("invalid token")
	}
	return claims, nil
}

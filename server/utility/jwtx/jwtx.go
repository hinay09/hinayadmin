// Package jwtx JWT 签发与解析。
package jwtx

import (
	"context"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/golang-jwt/jwt/v5"
	"time"
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

func loadConfig(ctx context.Context) *config {
	cfg := g.Cfg()
	secret, _ := cfg.Get(ctx, "jwt.secret", "hinay-admin-secret")
	expire, _ := cfg.Get(ctx, "jwt.expireSec", 86400)
	issuer, _ := cfg.Get(ctx, "jwt.issuer", "hinay-admin")
	return &config{
		Secret:    secret.String(),
		ExpireSec: expire.Int64(),
		Issuer:    issuer.String(),
	}
}

// Generate 签发 token, 返回 token 字符串与到期时间戳(秒)。
func Generate(ctx context.Context, userId uint64, username string) (string, int64, error) {
	c := loadConfig(ctx)
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
	c := loadConfig(ctx)
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

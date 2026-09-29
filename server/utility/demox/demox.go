// Package demox 演示环境开关。
//
// 开启后 (配置 demo.enable=true) 全局禁止修改/重置密码,
// 防止公开演示环境中的内置账号被改密导致演示不可用。
package demox

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/utility/xerror"
)

// Enabled 演示模式是否开启 (实时读取配置, 改配置无需重启)。
func Enabled(ctx context.Context) bool {
	v, err := g.Cfg().Get(ctx, "demo.enable", false)
	return err == nil && v.Bool()
}

// Guard 演示模式下禁止密码修改类操作, 修改/重置密码入口统一调用。
func Guard(ctx context.Context) error {
	if Enabled(ctx) {
		return xerror.New(xerror.CodeForbidden, "演示环境禁止修改密码 (配置 demo.enable=false 后开放)")
	}
	return nil
}

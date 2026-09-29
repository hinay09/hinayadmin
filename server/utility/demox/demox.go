// Package demox 演示环境开关。
//
// 开启后 (配置 demo.enable=true) 全局禁止修改/重置密码,
// 防止公开演示环境中的内置账号被改密导致演示不可用。
package demox

import (
	"context"
	"strconv"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/genv"

	"hinay.cn/admin/utility/xerror"
)

// parseBool 严格布尔解析: 仅认 true/false/1/0 等标准写法。
// gconv.Bool 对无法解析的非空字符串按"非空即真"处理,
// 未替换的 ${...} 占位符字面量会被其误判为 true, 故此处用 ParseBool 收紧。
func parseBool(s string) bool {
	b, err := strconv.ParseBool(strings.TrimSpace(s))
	return err == nil && b
}

// Enabled 演示模式是否开启。
// 优先读环境变量 DEMO_MODE (Docker 部署由 compose 注入, 进程启动即生效,
// 不依赖 entrypoint 的一次性占位符替换, 改 .env 后重建容器即可);
// 未设置时回落到配置 demo.enable。
func Enabled(ctx context.Context) bool {
	if env := genv.Get("DEMO_MODE").String(); strings.TrimSpace(env) != "" {
		return parseBool(env)
	}
	v, err := g.Cfg().Get(ctx, "demo.enable", false)
	if err != nil || v == nil {
		return false
	}
	switch t := v.Val().(type) {
	case bool:
		return t
	case string:
		return parseBool(t)
	default:
		return false
	}
}

// Guard 演示模式下禁止密码修改类操作, 修改/重置密码入口统一调用。
func Guard(ctx context.Context) error {
	if Enabled(ctx) {
		return xerror.New(xerror.CodeForbidden, "演示环境禁止修改密码 (配置 demo.enable=false 后开放)")
	}
	return nil
}

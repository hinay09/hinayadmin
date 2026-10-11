// Package tenantx 多租户总开关 (config.yaml multiTenant.enable) 的原子缓存读取。
//
// 开关语义 (docs/multi-tenant.md §9):
//   - 关闭 (默认): tenantorm 恒注入平台租户 tenant_id=1 (关闭 = 全部限定平台租户,
//     隔离兜底不失守), 登录不要求租户标识、IsAdmin 保持旧语义, 单租户老部署升级零感知;
//   - 开启: 行级隔离注入按 ctx 租户生效, 修改后需重启 (与 gencode.enable/demo.enable 文件开关先例一致)。
//
// 注意: 开关不放 sys_config —— 注入器读 sys_config 会鸡生蛋递归 (§9 修正)。
// 包内 atomic 缓存, 仅进程启动后首次读取配置, 每查询开销可忽略, 改配置需重启生效。
package tenantx

import (
	"context"
	"sync/atomic"

	"github.com/gogf/gf/v2/frame/g"
)

var (
	enabled     atomic.Bool
	initialized atomic.Bool
)

// Enabled 多租户是否启用 (首次调用读 config.yaml multiTenant.enable, 默认 false)。
func Enabled(ctx context.Context) bool {
	if initialized.Load() {
		return enabled.Load()
	}
	v, err := g.Cfg().Get(ctx, "multiTenant.enable", false)
	if err != nil || v == nil {
		enabled.Store(false)
	} else {
		enabled.Store(v.Bool())
	}
	initialized.Store(true)
	return enabled.Load()
}

// SetEnabledForTest 测试专用: 显式设置开关状态 (不读配置文件)。
func SetEnabledForTest(on bool) {
	enabled.Store(on)
	initialized.Store(true)
}

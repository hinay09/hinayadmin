// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	v1 "hinay.cn/admin/api/auth/v1"
)

type (
	ITwoFactor interface {
		// Enabled 用户是否已开启两步验证。
		//
		// 可选性契约 (fail-open): 两步验证是用户自选项, 本查询的任何失败 (含部署后未执行
		// 0011_totp.sql 建表) 一律按"未开启"降级, 绝不阻断未 opt-in 用户的登录与个人中心;
		// 仅告警一次提示补跑升级脚本。仅当绑定记录存在且 enabled=1 时才返回 true,
		// 即: 未绑定用户的所有既有流程与无本功能时逐字节一致。
		Enabled(ctx context.Context, userId uint64) bool
		// Setup 生成/重置当前用户的绑定密钥 (待验证状态)。
		// 重复调用会覆盖旧密钥; 已启用的用户须先解绑再重新绑定。
		// 演示环境禁止: 共享演示账号一旦绑定, 其他演示用户将无法通过两步验证登录。
		Setup(ctx context.Context, req *v1.TotpSetupReq) (res *v1.TotpSetupRes, err error)
		// Enable 确认绑定: 校验动态码通过后正式生效, 同时消费该码 (计入 last_step)。
		Enable(ctx context.Context, req *v1.TotpEnableReq) (res *v1.TotpEnableRes, err error)
		// Disable 解绑: 须提供当前有效动态码; 物理删除记录 (uk_user_id 与软删残留行冲突, 故不走软删)。
		Disable(ctx context.Context, req *v1.TotpDisableReq) (res *v1.TotpDisableRes, err error)
		// VerifyLogin 登录第二步动态码校验, 命中后推进 last_step 防重放。
		VerifyLogin(ctx context.Context, userId uint64, code string) error
	}
)

var (
	localTwoFactor ITwoFactor
)

func TwoFactor() ITwoFactor {
	if localTwoFactor == nil {
		panic("implement not found for interface ITwoFactor, forgot register?")
	}
	return localTwoFactor
}

func RegisterTwoFactor(i ITwoFactor) {
	localTwoFactor = i
}

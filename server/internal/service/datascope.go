// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"hinay.cn/admin/internal/model"
)

type (
	IDataScope interface {
		// OrgScope 计算当前登录用户的组织数据范围。
		// admin 角色恒为全部 (与 Casbin 中间件的全局放行语义一致)。
		OrgScope(ctx context.Context) (*model.OrgScope, error)
		// Apply 将数据范围套用到查询 (orgColumn/selfColumn 只允许来自调用方代码,
		// 不允许来自外部输入, 此处直接拼接进 SQL)。
		Apply(ctx context.Context, m *gdb.Model, orgColumn string, selfColumn string) (*gdb.Model, error)
	}
)

var (
	localDataScope IDataScope
)

func DataScope() IDataScope {
	if localDataScope == nil {
		panic("implement not found for interface IDataScope, forgot register?")
	}
	return localDataScope
}

func RegisterDataScope(i IDataScope) {
	localDataScope = i
}

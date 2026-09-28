// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

// Package service 组织数据权限服务接口。
package service

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"

	"hinay.cn/admin/internal/model"
)

type IDataScope interface {
	// OrgScope 计算当前登录用户的组织数据范围 (全部角色的并集)。
	OrgScope(ctx context.Context) (*model.OrgScope, error)
	// Apply 将组织数据范围套用到查询。
	// orgColumn: 业务表的组织字段名 (如 "org_id"); selfColumn: "仅本人" 比对字段 (如 "id"/"created_by")。
	// 两个列名只允许来自调用方代码, 不允许来自外部输入。
	Apply(ctx context.Context, m *gdb.Model, orgColumn, selfColumn string) (*gdb.Model, error)
}

var localDataScope IDataScope

func DataScope() IDataScope {
	if localDataScope == nil {
		panic("implement not found for interface IDataScope, forgot register?")
	}
	return localDataScope
}

func RegisterDataScope(i IDataScope) {
	localDataScope = i
}

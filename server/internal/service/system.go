// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
	"mime/multipart"

	v1 "hinay.cn/admin/api/system/v1"
)

type (
	IApi interface {
		// List 获取 API 列表（支持分组筛选、分页）。
		List(ctx context.Context, req *v1.ApiListReq) (res *v1.ApiListRes, err error)
		// All 全量 API（不分页，用于权限分配等场景）。
		All(ctx context.Context, _ *v1.ApiAllReq) (res *v1.ApiAllRes, err error)
		// Create 创建 API。
		Create(ctx context.Context, req *v1.ApiCreateReq) (res *v1.ApiCreateRes, err error)
		// Update 更新 API。
		Update(ctx context.Context, req *v1.ApiUpdateReq) (res *v1.ApiUpdateRes, err error)
		// Delete 删除 API。
		Delete(ctx context.Context, req *v1.ApiDeleteReq) (res *v1.ApiDeleteRes, err error)
	}
	IAuditLog interface {
		// List 分页列表。
		List(ctx context.Context, req *v1.AuditLogListReq) (res *v1.AuditLogListRes, err error)
	}
	IConfig interface {
		// List 分页列表。
		List(ctx context.Context, req *v1.ConfigListReq) (res *v1.ConfigListRes, err error)
		// All 获取所有启用的全局配置。
		All(ctx context.Context, _ *v1.ConfigAllReq) (res *v1.ConfigAllRes, err error)
		// Create 新增全局配置。
		Create(ctx context.Context, req *v1.ConfigCreateReq) (res *v1.ConfigCreateRes, err error)
		// Update 修改全局配置。
		Update(ctx context.Context, req *v1.ConfigUpdateReq) (res *v1.ConfigUpdateRes, err error)
		// Delete 删除全局配置（软删除）。
		Delete(ctx context.Context, req *v1.ConfigDeleteReq) (res *v1.ConfigDeleteRes, err error)
	}
	IDict interface {
		// ListByType 按类型获取字典数据项分页列表。
		ListByType(ctx context.Context, req *v1.DictDataListReq) (res *v1.DictDataListRes, err error)
		// Create 新增字典数据项。
		Create(ctx context.Context, req *v1.DictDataCreateReq) (res *v1.DictDataCreateRes, err error)
		// Update 修改字典数据项。
		Update(ctx context.Context, req *v1.DictDataUpdateReq) (res *v1.DictDataUpdateRes, err error)
		// Delete 删除字典数据项。
		Delete(ctx context.Context, req *v1.DictDataDeleteReq) (res *v1.DictDataDeleteRes, err error)
		// Sort 批量排序字典数据项。
		Sort(ctx context.Context, req *v1.DictDataSortReq) (res *v1.DictDataSortRes, err error)
		// All 获取所有启用的字典数据（按类型编码分组）。
		All(ctx context.Context, _ *v1.DictAllReq) (res *v1.DictAllRes, err error)
	}
	IDictType interface {
		// List 字典类型分页列表。
		List(ctx context.Context, req *v1.DictTypeListReq) (res *v1.DictTypeListRes, err error)
		// All 获取所有启用的字典类型。
		All(ctx context.Context, _ *v1.DictTypeAllReq) (res *v1.DictTypeAllRes, err error)
		// Create 新增字典类型。
		Create(ctx context.Context, req *v1.DictTypeCreateReq) (res *v1.DictTypeCreateRes, err error)
		// Update 修改字典类型。
		Update(ctx context.Context, req *v1.DictTypeUpdateReq) (res *v1.DictTypeUpdateRes, err error)
		// Delete 删除字典类型（有子项则拒绝删除）。
		Delete(ctx context.Context, req *v1.DictTypeDeleteReq) (res *v1.DictTypeDeleteRes, err error)
	}
	IFile interface {
		// List 分页列表。
		List(ctx context.Context, req *v1.FileListReq) (res *v1.FileListRes, err error)
		// Upload 上传文件。
		// 校验链: 扩展名白名单 -> 大小上限 -> 文件头内容嗅探(拒绝页面/脚本类真实内容)。
		Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader) (res *v1.FileUploadRes, err error)
		// Delete 删除文件（软删记录）。
		Delete(ctx context.Context, req *v1.FileDeleteReq) (res *v1.FileDeleteRes, err error)
	}
	IMenu interface {
		// List 扁平列表。
		List(ctx context.Context, req *v1.MenuListReq) (res *v1.MenuListRes, err error)
		// Tree 树形菜单(全部启用菜单, 不按权限过滤, 用于后台维护)。
		Tree(ctx context.Context, _ *v1.MenuTreeReq) (res *v1.MenuTreeRes, err error)
		// Detail 详情。
		Detail(ctx context.Context, req *v1.MenuDetailReq) (res *v1.MenuDetailRes, err error)
		// Create 新增。
		Create(ctx context.Context, req *v1.MenuCreateReq) (res *v1.MenuCreateRes, err error)
		// Update 修改。
		Update(ctx context.Context, req *v1.MenuUpdateReq) (res *v1.MenuUpdateRes, err error)
		// Delete 软删除 (含子节点检查)。
		Delete(ctx context.Context, req *v1.MenuDeleteReq) (res *v1.MenuDeleteRes, err error)
	}
	IOrg interface {
		// Tree 树形组织机构。
		Tree(ctx context.Context, _ *v1.OrgTreeReq) (res *v1.OrgTreeRes, err error)
		// List 扁平列表。
		List(ctx context.Context, req *v1.OrgListReq) (res *v1.OrgListRes, err error)
		// Detail 详情。
		Detail(ctx context.Context, req *v1.OrgDetailReq) (res *v1.OrgDetailRes, err error)
		// Create 新增。
		Create(ctx context.Context, req *v1.OrgCreateReq) (res *v1.OrgCreateRes, err error)
		// Update 修改。
		Update(ctx context.Context, req *v1.OrgUpdateReq) (res *v1.OrgUpdateRes, err error)
		// Delete 软删除 (含子节点检查)。
		Delete(ctx context.Context, req *v1.OrgDeleteReq) (res *v1.OrgDeleteRes, err error)
	}
	IRole interface {
		// List 分页列表。
		List(ctx context.Context, req *v1.RoleListReq) (res *v1.RoleListRes, err error)
		// All 全量。
		All(ctx context.Context, _ *v1.RoleAllReq) (res *v1.RoleAllRes, err error)
		// Detail 详情 + 已绑定菜单 ID 列表（从 Casbin 获取）。
		Detail(ctx context.Context, req *v1.RoleDetailReq) (res *v1.RoleDetailRes, err error)
		// Create 新增。
		Create(ctx context.Context, req *v1.RoleCreateReq) (res *v1.RoleCreateRes, err error)
		// Update 修改。
		Update(ctx context.Context, req *v1.RoleUpdateReq) (res *v1.RoleUpdateRes, err error)
		// Delete 删除 (软删 + 清理 Casbin 策略)。
		Delete(ctx context.Context, req *v1.RoleDeleteReq) (res *v1.RoleDeleteRes, err error)
		// AssignMenus 角色绑定菜单, 通过 Casbin 策略管理。
		AssignMenus(ctx context.Context, req *v1.RoleAssignMenusReq) (res *v1.RoleAssignMenusRes, err error)
		// GetMenus 获取角色已绑定的菜单 ID 列表（从 Casbin 获取）。
		GetMenus(ctx context.Context, req *v1.RoleGetMenusReq) (res *v1.RoleGetMenusRes, err error)
		// AssignApis 角色分配 API 权限。
		AssignApis(ctx context.Context, req *v1.RoleAssignApisReq) (res *v1.RoleAssignApisRes, err error)
		// GetApis 获取角色已分配的 API 权限。
		GetApis(ctx context.Context, req *v1.RoleGetApisReq) (res *v1.RoleGetApisRes, err error)
	}
	IUser interface {
		// List 分页列表。
		List(ctx context.Context, req *v1.UserListReq) (res *v1.UserListRes, err error)
		// Detail 详情。
		Detail(ctx context.Context, req *v1.UserDetailReq) (res *v1.UserDetailRes, err error)
		// Create 新增。
		Create(ctx context.Context, req *v1.UserCreateReq) (res *v1.UserCreateRes, err error)
		// Update 修改。
		Update(ctx context.Context, req *v1.UserUpdateReq) (res *v1.UserUpdateRes, err error)
		// Delete 软删除 + 清理 Casbin g 策略。
		Delete(ctx context.Context, req *v1.UserDeleteReq) (res *v1.UserDeleteRes, err error)
		// ResetPwd 重置密码。
		ResetPwd(ctx context.Context, req *v1.UserResetPwdReq) (res *v1.UserResetPwdRes, err error)
	}
)

var (
	localApi      IApi
	localAuditLog IAuditLog
	localConfig   IConfig
	localDict     IDict
	localDictType IDictType
	localFile     IFile
	localMenu     IMenu
	localOrg      IOrg
	localRole     IRole
	localUser     IUser
)

func Api() IApi {
	if localApi == nil {
		panic("implement not found for interface IApi, forgot register?")
	}
	return localApi
}

func RegisterApi(i IApi) {
	localApi = i
}

func AuditLog() IAuditLog {
	if localAuditLog == nil {
		panic("implement not found for interface IAuditLog, forgot register?")
	}
	return localAuditLog
}

func RegisterAuditLog(i IAuditLog) {
	localAuditLog = i
}

func Config() IConfig {
	if localConfig == nil {
		panic("implement not found for interface IConfig, forgot register?")
	}
	return localConfig
}

func RegisterConfig(i IConfig) {
	localConfig = i
}

func Dict() IDict {
	if localDict == nil {
		panic("implement not found for interface IDict, forgot register?")
	}
	return localDict
}

func RegisterDict(i IDict) {
	localDict = i
}

func DictType() IDictType {
	if localDictType == nil {
		panic("implement not found for interface IDictType, forgot register?")
	}
	return localDictType
}

func RegisterDictType(i IDictType) {
	localDictType = i
}

func File() IFile {
	if localFile == nil {
		panic("implement not found for interface IFile, forgot register?")
	}
	return localFile
}

func RegisterFile(i IFile) {
	localFile = i
}

func Menu() IMenu {
	if localMenu == nil {
		panic("implement not found for interface IMenu, forgot register?")
	}
	return localMenu
}

func RegisterMenu(i IMenu) {
	localMenu = i
}

func Org() IOrg {
	if localOrg == nil {
		panic("implement not found for interface IOrg, forgot register?")
	}
	return localOrg
}

func RegisterOrg(i IOrg) {
	localOrg = i
}

func Role() IRole {
	if localRole == nil {
		panic("implement not found for interface IRole, forgot register?")
	}
	return localRole
}

func RegisterRole(i IRole) {
	localRole = i
}

func User() IUser {
	if localUser == nil {
		panic("implement not found for interface IUser, forgot register?")
	}
	return localUser
}

func RegisterUser(i IUser) {
	localUser = i
}

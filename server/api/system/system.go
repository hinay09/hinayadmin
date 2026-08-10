// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package system

import (
	"context"

	"hinay.cn/admin/api/system/v1"
)

type ISystemV1 interface {
	ApiList(ctx context.Context, req *v1.ApiListReq) (res *v1.ApiListRes, err error)
	ApiAll(ctx context.Context, req *v1.ApiAllReq) (res *v1.ApiAllRes, err error)
	ApiCreate(ctx context.Context, req *v1.ApiCreateReq) (res *v1.ApiCreateRes, err error)
	ApiUpdate(ctx context.Context, req *v1.ApiUpdateReq) (res *v1.ApiUpdateRes, err error)
	ApiDelete(ctx context.Context, req *v1.ApiDeleteReq) (res *v1.ApiDeleteRes, err error)
	AuditLogList(ctx context.Context, req *v1.AuditLogListReq) (res *v1.AuditLogListRes, err error)
	ConfigList(ctx context.Context, req *v1.ConfigListReq) (res *v1.ConfigListRes, err error)
	ConfigAll(ctx context.Context, req *v1.ConfigAllReq) (res *v1.ConfigAllRes, err error)
	ConfigCreate(ctx context.Context, req *v1.ConfigCreateReq) (res *v1.ConfigCreateRes, err error)
	ConfigUpdate(ctx context.Context, req *v1.ConfigUpdateReq) (res *v1.ConfigUpdateRes, err error)
	ConfigDelete(ctx context.Context, req *v1.ConfigDeleteReq) (res *v1.ConfigDeleteRes, err error)
	DictTypeList(ctx context.Context, req *v1.DictTypeListReq) (res *v1.DictTypeListRes, err error)
	DictTypeAll(ctx context.Context, req *v1.DictTypeAllReq) (res *v1.DictTypeAllRes, err error)
	DictTypeCreate(ctx context.Context, req *v1.DictTypeCreateReq) (res *v1.DictTypeCreateRes, err error)
	DictTypeUpdate(ctx context.Context, req *v1.DictTypeUpdateReq) (res *v1.DictTypeUpdateRes, err error)
	DictTypeDelete(ctx context.Context, req *v1.DictTypeDeleteReq) (res *v1.DictTypeDeleteRes, err error)
	DictDataList(ctx context.Context, req *v1.DictDataListReq) (res *v1.DictDataListRes, err error)
	DictDataCreate(ctx context.Context, req *v1.DictDataCreateReq) (res *v1.DictDataCreateRes, err error)
	DictDataUpdate(ctx context.Context, req *v1.DictDataUpdateReq) (res *v1.DictDataUpdateRes, err error)
	DictDataDelete(ctx context.Context, req *v1.DictDataDeleteReq) (res *v1.DictDataDeleteRes, err error)
	DictDataSort(ctx context.Context, req *v1.DictDataSortReq) (res *v1.DictDataSortRes, err error)
	DictAll(ctx context.Context, req *v1.DictAllReq) (res *v1.DictAllRes, err error)
	FileList(ctx context.Context, req *v1.FileListReq) (res *v1.FileListRes, err error)
	FileUpload(ctx context.Context, req *v1.FileUploadReq) (res *v1.FileUploadRes, err error)
	FileDelete(ctx context.Context, req *v1.FileDeleteReq) (res *v1.FileDeleteRes, err error)
	MenuList(ctx context.Context, req *v1.MenuListReq) (res *v1.MenuListRes, err error)
	MenuTree(ctx context.Context, req *v1.MenuTreeReq) (res *v1.MenuTreeRes, err error)
	MenuDetail(ctx context.Context, req *v1.MenuDetailReq) (res *v1.MenuDetailRes, err error)
	MenuCreate(ctx context.Context, req *v1.MenuCreateReq) (res *v1.MenuCreateRes, err error)
	MenuUpdate(ctx context.Context, req *v1.MenuUpdateReq) (res *v1.MenuUpdateRes, err error)
	MenuDelete(ctx context.Context, req *v1.MenuDeleteReq) (res *v1.MenuDeleteRes, err error)
	OrgTree(ctx context.Context, req *v1.OrgTreeReq) (res *v1.OrgTreeRes, err error)
	OrgList(ctx context.Context, req *v1.OrgListReq) (res *v1.OrgListRes, err error)
	OrgDetail(ctx context.Context, req *v1.OrgDetailReq) (res *v1.OrgDetailRes, err error)
	OrgCreate(ctx context.Context, req *v1.OrgCreateReq) (res *v1.OrgCreateRes, err error)
	OrgUpdate(ctx context.Context, req *v1.OrgUpdateReq) (res *v1.OrgUpdateRes, err error)
	OrgDelete(ctx context.Context, req *v1.OrgDeleteReq) (res *v1.OrgDeleteRes, err error)
	RoleList(ctx context.Context, req *v1.RoleListReq) (res *v1.RoleListRes, err error)
	RoleAll(ctx context.Context, req *v1.RoleAllReq) (res *v1.RoleAllRes, err error)
	RoleDetail(ctx context.Context, req *v1.RoleDetailReq) (res *v1.RoleDetailRes, err error)
	RoleCreate(ctx context.Context, req *v1.RoleCreateReq) (res *v1.RoleCreateRes, err error)
	RoleUpdate(ctx context.Context, req *v1.RoleUpdateReq) (res *v1.RoleUpdateRes, err error)
	RoleDelete(ctx context.Context, req *v1.RoleDeleteReq) (res *v1.RoleDeleteRes, err error)
	RoleAssignMenus(ctx context.Context, req *v1.RoleAssignMenusReq) (res *v1.RoleAssignMenusRes, err error)
	RoleGetMenus(ctx context.Context, req *v1.RoleGetMenusReq) (res *v1.RoleGetMenusRes, err error)
	RoleAssignApis(ctx context.Context, req *v1.RoleAssignApisReq) (res *v1.RoleAssignApisRes, err error)
	RoleGetApis(ctx context.Context, req *v1.RoleGetApisReq) (res *v1.RoleGetApisRes, err error)
	UserList(ctx context.Context, req *v1.UserListReq) (res *v1.UserListRes, err error)
	UserDetail(ctx context.Context, req *v1.UserDetailReq) (res *v1.UserDetailRes, err error)
	UserCreate(ctx context.Context, req *v1.UserCreateReq) (res *v1.UserCreateRes, err error)
	UserUpdate(ctx context.Context, req *v1.UserUpdateReq) (res *v1.UserUpdateRes, err error)
	UserDelete(ctx context.Context, req *v1.UserDeleteReq) (res *v1.UserDeleteRes, err error)
	UserResetPwd(ctx context.Context, req *v1.UserResetPwdReq) (res *v1.UserResetPwdRes, err error)
}

// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish to manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	v1 "hinay.cn/admin/api/post/v1"
)

type (
	IPost interface {
		// PostList 岗位分页列表 (含成员数)。
		PostList(ctx context.Context, in *v1.PostListReq) (res *v1.PostListRes, err error)
		// PostAll 启用岗位全量。
		PostAll(ctx context.Context, in *v1.PostAllReq) (res *v1.PostAllRes, err error)
		// PostCreate 新增岗位 (编码唯一)。
		PostCreate(ctx context.Context, in *v1.PostCreateReq) (res *v1.PostCreateRes, err error)
		// PostUpdate 修改岗位。
		PostUpdate(ctx context.Context, in *v1.PostUpdateReq) (res *v1.PostUpdateRes, err error)
		// PostDelete 删除岗位 (软删, 同时清理挂岗关系)。
		PostDelete(ctx context.Context, in *v1.PostDeleteReq) (res *v1.PostDeleteRes, err error)
		// PostMemberList 岗位成员列表。
		PostMemberList(ctx context.Context, in *v1.PostMemberListReq) (res *v1.PostMemberListRes, err error)
		// PostMemberAdd 添加岗位成员。
		PostMemberAdd(ctx context.Context, in *v1.PostMemberAddReq) (res *v1.PostMemberAddRes, err error)
		// PostMemberRemove 移除岗位成员。
		PostMemberRemove(ctx context.Context, in *v1.PostMemberRemoveReq) (res *v1.PostMemberRemoveRes, err error)
	}
)

var (
	localPost IPost
)

func Post() IPost {
	if localPost == nil {
		panic("implement not found for interface IPost, forgot register?")
	}
	return localPost
}

func RegisterPost(i IPost) {
	localPost = i
}

// Package post 岗位管理控制器 (薄透传)。
package post

import (
	"context"

	"hinay.cn/admin/api/post/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) PostList(ctx context.Context, req *v1.PostListReq) (res *v1.PostListRes, err error) {
	return service.Post().PostList(ctx, req)
}

func (c *ControllerV1) PostAll(ctx context.Context, req *v1.PostAllReq) (res *v1.PostAllRes, err error) {
	return service.Post().PostAll(ctx, req)
}

func (c *ControllerV1) PostCreate(ctx context.Context, req *v1.PostCreateReq) (res *v1.PostCreateRes, err error) {
	return service.Post().PostCreate(ctx, req)
}

func (c *ControllerV1) PostUpdate(ctx context.Context, req *v1.PostUpdateReq) (res *v1.PostUpdateRes, err error) {
	return service.Post().PostUpdate(ctx, req)
}

func (c *ControllerV1) PostDelete(ctx context.Context, req *v1.PostDeleteReq) (res *v1.PostDeleteRes, err error) {
	return service.Post().PostDelete(ctx, req)
}

func (c *ControllerV1) PostMemberList(ctx context.Context, req *v1.PostMemberListReq) (res *v1.PostMemberListRes, err error) {
	return service.Post().PostMemberList(ctx, req)
}

func (c *ControllerV1) PostMemberAdd(ctx context.Context, req *v1.PostMemberAddReq) (res *v1.PostMemberAddRes, err error) {
	return service.Post().PostMemberAdd(ctx, req)
}

func (c *ControllerV1) PostMemberRemove(ctx context.Context, req *v1.PostMemberRemoveReq) (res *v1.PostMemberRemoveRes, err error) {
	return service.Post().PostMemberRemove(ctx, req)
}

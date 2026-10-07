// Package post 岗位管理业务逻辑。
package post

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"

	v1 "hinay.cn/admin/api/post/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model/entity"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

func init() {
	service.RegisterPost(&sPost{})
}

type sPost struct{}

// postItem 实体转条目 (含成员数)。
func postItem(ctx context.Context, p *entity.SysPost) *v1.PostItem {
	cnt, _ := dao.SysUserPost.Ctx(ctx).Where("post_id", p.Id).Count()
	return &v1.PostItem{
		Id: p.Id, PostCode: p.PostCode, PostName: p.PostName, PostKind: p.PostKind,
		Sort: p.Sort, Status: p.Status, Remark: p.Remark, Members: int64(cnt), UpdateAt: p.UpdatedAt,
	}
}

// PostList 岗位分页列表。
func (s *sPost) PostList(ctx context.Context, in *v1.PostListReq) (res *v1.PostListRes, err error) {
	q := dao.SysPost.Ctx(ctx).Where("deleted_at IS NULL")
	if in.Keyword != "" {
		kw := "%" + strings.TrimSpace(in.Keyword) + "%"
		q = q.Where("(post_code LIKE ? OR post_name LIKE ?)", kw, kw)
	}
	if in.Status != nil {
		q = q.Where("status", *in.Status)
	}
	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var rows []*entity.SysPost
	if err = q.Order("sort ASC, id ASC").Page(in.Page, in.PageSize).Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	list := make([]*v1.PostItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, postItem(ctx, r))
	}
	page := response.Page(list, int64(total), in.Page, in.PageSize)
	return (*v1.PostListRes)(&page), nil
}

// PostAll 启用岗位全量。
func (s *sPost) PostAll(ctx context.Context, in *v1.PostAllReq) (res *v1.PostAllRes, err error) {
	var rows []*entity.SysPost
	if err = dao.SysPost.Ctx(ctx).Where("status", 1).Where("deleted_at IS NULL").
		Order("sort ASC, id ASC").Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	list := make([]*v1.PostItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, &v1.PostItem{
			Id: r.Id, PostCode: r.PostCode, PostName: r.PostName, PostKind: r.PostKind,
			Sort: r.Sort, Status: r.Status,
		})
	}
	return &v1.PostAllRes{List: list}, nil
}

// PostCreate 新增岗位。
func (s *sPost) PostCreate(ctx context.Context, in *v1.PostCreateReq) (res *v1.PostCreateRes, err error) {
	if s.codeExists(ctx, strings.TrimSpace(in.PostCode), 0) {
		return nil, xerror.New(xerror.CodeParamInvalid, "岗位编码已存在")
	}
	id, err := dao.SysPost.Ctx(ctx).Data(g.Map{
		"post_code": strings.TrimSpace(in.PostCode),
		"post_name": in.PostName,
		"post_kind": in.PostKind,
		"sort":      in.Sort,
		"status":    in.Status,
		"remark":    in.Remark,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.PostCreateRes{Id: uint64(id)}, nil
}

// PostUpdate 修改岗位。
func (s *sPost) PostUpdate(ctx context.Context, in *v1.PostUpdateReq) (res *v1.PostUpdateRes, err error) {
	if s.codeExists(ctx, strings.TrimSpace(in.PostCode), in.Id) {
		return nil, xerror.New(xerror.CodeParamInvalid, "岗位编码已存在")
	}
	if _, err = dao.SysPost.Ctx(ctx).Where("id", in.Id).Data(g.Map{
		"post_code": strings.TrimSpace(in.PostCode),
		"post_name": in.PostName,
		"post_kind": in.PostKind,
		"sort":      in.Sort,
		"status":    in.Status,
		"remark":    in.Remark,
	}).Update(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.PostUpdateRes{}, nil
}

// PostDelete 删除岗位 (软删 + 清理挂岗关系)。
func (s *sPost) PostDelete(ctx context.Context, in *v1.PostDeleteReq) (res *v1.PostDeleteRes, err error) {
	if _, err = dao.SysPost.Ctx(ctx).Where("id", in.Id).Delete(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if _, err = dao.SysUserPost.Ctx(ctx).Where("post_id", in.Id).Delete(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.PostDeleteRes{}, nil
}

// codeExists 编码唯一性检查 (excludeId 排除自身)。
func (s *sPost) codeExists(ctx context.Context, code string, excludeId uint64) bool {
	if code == "" {
		return false
	}
	cnt, err := dao.SysPost.Ctx(ctx).Where("post_code", code).
		Where("deleted_at IS NULL").WhereNot("id", excludeId).Count()
	return err == nil && cnt > 0
}

// ============================================================
// 岗位成员
// ============================================================

// PostMemberList 岗位成员列表。
func (s *sPost) PostMemberList(ctx context.Context, in *v1.PostMemberListReq) (res *v1.PostMemberListRes, err error) {
	var rels []*entity.SysUserPost
	if err = dao.SysUserPost.Ctx(ctx).Where("post_id", in.Id).Order("id DESC").Scan(&rels); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	list := make([]*v1.PostMemberItem, 0, len(rels))
	for _, r := range rels {
		item := &v1.PostMemberItem{Id: r.Id, UserId: r.UserId, OrgId: r.OrgId, CreateAt: r.CreatedAt}
		var u *entity.SysUser
		if e := dao.SysUser.Ctx(ctx).Where("id", r.UserId).Where("deleted_at IS NULL").Scan(&u); e == nil && u != nil {
			item.UserName = pickName(u.Nickname, u.Username)
		}
		if r.OrgId > 0 {
			var org *entity.SysOrg
			if e := dao.SysOrg.Ctx(ctx).Where("id", r.OrgId).Where("deleted_at IS NULL").Scan(&org); e == nil && org != nil {
				item.OrgName = org.Name
			}
		}
		list = append(list, item)
	}
	return &v1.PostMemberListRes{List: list}, nil
}

// PostMemberAdd 添加岗位成员 (同一 用户×岗位×组织 唯一)。
func (s *sPost) PostMemberAdd(ctx context.Context, in *v1.PostMemberAddReq) (res *v1.PostMemberAddRes, err error) {
	var post *entity.SysPost
	if err = dao.SysPost.Ctx(ctx).Where("id", in.Id).Where("deleted_at IS NULL").Scan(&post); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if post == nil {
		return nil, xerror.New(xerror.CodeNotFound, "岗位不存在")
	}
	var u *entity.SysUser
	if err = dao.SysUser.Ctx(ctx).Where("id", in.UserId).Where("deleted_at IS NULL").Scan(&u); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if u == nil {
		return nil, xerror.New(xerror.CodeParamInvalid, "用户不存在")
	}
	if in.OrgId > 0 {
		cnt, _ := dao.SysOrg.Ctx(ctx).Where("id", in.OrgId).Where("deleted_at IS NULL").Count()
		if cnt == 0 {
			return nil, xerror.New(xerror.CodeParamInvalid, "组织不存在")
		}
	}
	dup, _ := dao.SysUserPost.Ctx(ctx).
		Where("user_id", in.UserId).Where("post_id", in.Id).Where("org_id", in.OrgId).Count()
	if dup > 0 {
		return nil, xerror.New(xerror.CodeParamInvalid, "该用户已在此岗位(同组织)")
	}
	id, err := dao.SysUserPost.Ctx(ctx).Data(g.Map{
		"user_id": in.UserId, "post_id": in.Id, "org_id": in.OrgId,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.PostMemberAddRes{Id: uint64(id)}, nil
}

// PostMemberRemove 移除岗位成员。
func (s *sPost) PostMemberRemove(ctx context.Context, in *v1.PostMemberRemoveReq) (res *v1.PostMemberRemoveRes, err error) {
	if _, err = dao.SysUserPost.Ctx(ctx).Where("id", in.RelId).Delete(); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.PostMemberRemoveRes{}, nil
}

// pickName 昵称优先。
func pickName(nickname, username string) string {
	if nickname != "" {
		return nickname
	}
	return username
}

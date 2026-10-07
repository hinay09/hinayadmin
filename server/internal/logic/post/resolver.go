// Package post — 审批引擎的审批人解析辅助 (按岗位找人 / 部门主管向上递归)。
package post

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model/entity"
)

// PostUser 解析出的岗位人员。
type PostUser struct {
	Id   uint64
	Name string
}

// UsersOfPost 取挂指定岗位的启用用户 (含"不限定组织"的挂岗), 供"指定岗位"审批人类型使用。
func UsersOfPost(ctx context.Context, postId uint64) ([]PostUser, error) {
	var rels []*entity.SysUserPost
	if err := dao.SysUserPost.Ctx(ctx).Where("post_id", postId).Scan(&rels); err != nil {
		return nil, err
	}
	if len(rels) == 0 {
		return nil, nil
	}
	ids := make([]uint64, 0, len(rels))
	for _, r := range rels {
		ids = append(ids, r.UserId)
	}
	var users []*entity.SysUser
	if err := dao.SysUser.Ctx(ctx).
		WhereIn("id", ids).Where("status", 1).Where("deleted_at IS NULL").
		Order("id ASC").Scan(&users); err != nil {
		return nil, err
	}
	seen := map[uint64]bool{}
	out := make([]PostUser, 0, len(users))
	for _, u := range users {
		if seen[u.Id] {
			continue
		}
		seen[u.Id] = true
		out = append(out, PostUser{Id: u.Id, Name: pickName(u.Nickname, u.Username)})
	}
	return out, nil
}

// FindSuperior 解析"部门主管": 从发起人所在组织起, 逐级向上找挂有启用主管岗
// (sys_post.post_kind=2) 的启用用户 (跳过发起人自己, 防止自己审自己), 命中即返回。
// 典型链路: 员工 → 组长 → 部门经理 → 总经理; 到根仍无则报错 (宁失败不误发)。
func FindSuperior(ctx context.Context, userId uint64) (PostUser, error) {
	var u *entity.SysUser
	if err := dao.SysUser.Ctx(ctx).Where("id", userId).Where("deleted_at IS NULL").Scan(&u); err != nil {
		return PostUser{}, err
	}
	if u == nil {
		return PostUser{}, fmt.Errorf("发起人用户不存在")
	}
	if u.OrgId == 0 {
		return PostUser{}, fmt.Errorf("发起人未设置所属组织, 无法解析部门主管")
	}

	orgId := u.OrgId
	for hop := 0; orgId > 0 && hop < 20; hop++ {
		var org *entity.SysOrg
		if err := dao.SysOrg.Ctx(ctx).Where("id", orgId).Where("deleted_at IS NULL").Scan(&org); err != nil {
			return PostUser{}, err
		}
		if org == nil {
			break
		}
		if hit, ok := leaderInOrg(ctx, orgId, userId); ok {
			return hit, nil
		}
		orgId = org.ParentId
	}
	return PostUser{}, fmt.Errorf("发起人所在组织及上级组织均无人挂主管岗 (在岗位管理中创建主管岗并指派成员)")
}

// leaderInOrg 组织内挂主管岗的启用用户 (按用户 id 取首位, 跳过发起人)。
func leaderInOrg(ctx context.Context, orgId, excludeUserId uint64) (PostUser, bool) {
	// 启用的主管岗
	postIds, err := dao.SysPost.Ctx(ctx).
		Where("post_kind", 2).Where("status", 1).Where("deleted_at IS NULL").
		Array("id")
	if err != nil || len(postIds) == 0 {
		return PostUser{}, false
	}
	vals, err := dao.SysUserPost.Ctx(ctx).
		Where("org_id", orgId).WhereIn("post_id", postIds).
		Array("user_id")
	if err != nil || len(vals) == 0 {
		return PostUser{}, false
	}
	ids := make([]uint64, 0, len(vals))
	for _, v := range vals {
		id := g.NewVar(v).Uint64()
		if id > 0 && id != excludeUserId {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return PostUser{}, false
	}
	var users []*entity.SysUser
	if err := dao.SysUser.Ctx(ctx).
		WhereIn("id", ids).Where("status", 1).Where("deleted_at IS NULL").
		Order("id ASC").Limit(1).Scan(&users); err != nil || len(users) == 0 {
		return PostUser{}, false
	}
	return PostUser{Id: users[0].Id, Name: pickName(users[0].Nickname, users[0].Username)}, true
}

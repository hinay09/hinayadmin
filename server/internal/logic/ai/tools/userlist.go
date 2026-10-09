// get_user_list: 系统人员列表 (数据权限过滤, 通讯录字段)。
package tools

import (
	"context"
	"fmt"
	"strings"

	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/service"
)

// 返回条数: 默认/上限 (防止把全量人员灌进模型上下文, 也避免超长回填)。
const (
	userToolDefaultLimit = 10
	userToolMaxLimit     = 50
)

// userListArgs 入参 (InferTool 据此反射出模型可见的参数 schema)。
type userListArgs struct {
	Keyword string `json:"keyword,omitempty" jsonschema_description:"搜索关键词, 模糊匹配 编码(登录账号)/姓名/手机号/邮箱; 空则按数据权限列出全部人员"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"返回条数上限, 默认 10, 最大 50"`
}

// userItem 一名人员的对外字段 (仅通讯录信息, 不暴露 ID/状态/登录痕迹等内部字段)。
type userItem struct {
	Code  string `json:"code"`            // 编码 (登录账号)
	Name  string `json:"name"`            // 姓名 (昵称)
	Phone string `json:"phone,omitempty"` // 手机号
	Email string `json:"email,omitempty"` // 邮箱
	Org   string `json:"org,omitempty"`   // 所属组织机构名
}

// userListResult 返回值 (InferTool 序列化为 JSON 结果文本)。
type userListResult struct {
	Total int        `json:"total"` // 数据权限范围内匹配总人数
	Count int        `json:"count"` // 本次实际返回条数 (total 更大时已按 limit 截断)
	Users []userItem `json:"users"` // 人员列表 (按账号编码排序)
}

// runUserList 查询人员列表。数据权限与「用户管理」列表同源同口径:
// service.DataScope().Apply 按当前用户角色的数据范围 (全部/自定义/本部门[及以下]/仅本人)
// 过滤可见行 — AI 工具不构成新的越权通道; 无任何可见范围时 fail-close 返回空集。
func runUserList(ctx context.Context, args userListArgs) (*userListResult, error) {
	limit := clampUserLimit(args.Limit)

	q := dao.SysUser.Ctx(ctx).
		Where("deleted_at IS NULL").
		Where("status", consts.StatusEnabled) // 禁用账号多为离职/停用, 通讯录场景不返回
	if kw := strings.TrimSpace(args.Keyword); kw != "" {
		like := "%" + kw + "%"
		// 括号分组: 避免关键词 OR 条件逃逸出 deleted_at/status 过滤
		q = q.Where("(username LIKE ? OR nickname LIKE ? OR phone LIKE ? OR email LIKE ?)",
			like, like, like, like)
	}
	q, err := service.DataScope().Apply(ctx, q, "org_id", "id")
	if err != nil {
		return nil, fmt.Errorf("数据权限计算失败: %w", err)
	}

	// 注意: Count 不能带 Fields — GoFrame 会拼成 COUNT(col1,col2,...) 多列聚合, MySQL/MariaDB 均为语法错误;
	// 与 system/user.go 列表同款写法: 先无字段 Count (生成 COUNT(1)), 再对取行查询限定字段。
	total, err := q.Ctx(ctx).Count()
	if err != nil {
		return nil, err
	}
	rows, err := q.Ctx(ctx).
		Fields("username, nickname, phone, email, org_id").
		Order("id ASC").Limit(limit).All()
	if err != nil {
		return nil, err
	}

	// 组织机构名: 一次批量查询 (避免逐行回表)
	orgIds := make([]uint64, 0, len(rows))
	seen := map[uint64]struct{}{}
	for _, r := range rows {
		id := r["org_id"].Uint64()
		if id > 0 {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				orgIds = append(orgIds, id)
			}
		}
	}
	orgNames := map[uint64]string{}
	if len(orgIds) > 0 {
		orgs, oerr := dao.SysOrg.Ctx(ctx).
			Fields("id, name").
			WhereIn("id", orgIds).
			Where("deleted_at IS NULL").
			All()
		if oerr != nil {
			return nil, oerr
		}
		for _, o := range orgs {
			orgNames[o["id"].Uint64()] = strings.TrimSpace(o["name"].String())
		}
	}

	users := make([]userItem, 0, len(rows))
	for _, r := range rows {
		users = append(users, userItem{
			Code:  strings.TrimSpace(r["username"].String()),
			Name:  strings.TrimSpace(r["nickname"].String()),
			Phone: strings.TrimSpace(r["phone"].String()),
			Email: strings.TrimSpace(r["email"].String()),
			Org:   orgNames[r["org_id"].Uint64()],
		})
	}
	return &userListResult{Total: total, Count: len(users), Users: users}, nil
}

// clampUserLimit 归一化返回条数 (0/负数 → 默认; 超上限 → 上限)。
func clampUserLimit(n int) int {
	if n <= 0 {
		return userToolDefaultLimit
	}
	if n > userToolMaxLimit {
		return userToolMaxLimit
	}
	return n
}

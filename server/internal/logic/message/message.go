// Package message 消息通知业务逻辑。
package message

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	v1 "hinay.cn/admin/api/message/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/logic/casbinx"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

type sMessage struct{}

func init() {
	service.RegisterMessage(NewMessage())
}

func NewMessage() *sMessage {
	return &sMessage{}
}

// List 管理员视角的消息分页列表。
func (s *sMessage) List(ctx context.Context, in *v1.MessageListReq) (res *v1.MessageListRes, err error) {
	q := dao.BizMessage.Ctx(ctx).Where("deleted_at IS NULL")
	if in.Type != nil {
		q = q.Where("type", *in.Type)
	}
	if in.Level != nil {
		q = q.Where("level", *in.Level)
	}
	if in.Status != nil {
		q = q.Where("status", *in.Status)
	}
	if in.Keyword != "" {
		q = q.WhereLike("title", "%"+strings.TrimSpace(in.Keyword)+"%")
	}
	total, err := q.Ctx(ctx).Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var rows []*model.BizMessage
	if err = q.Ctx(ctx).Page(in.Page, in.PageSize).Order("id DESC").Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	list, err := decorateMessages(ctx, rows, 0)
	if err != nil {
		return nil, err
	}
	page := response.Page(list, int64(total), in.Page, in.PageSize)
	r := v1.MessageListRes(page)
	return &r, nil
}

// Detail 管理员视角详情。
func (s *sMessage) Detail(ctx context.Context, in *v1.MessageDetailReq) (res *v1.MessageDetailRes, err error) {
	var m *model.BizMessage
	if err = dao.BizMessage.Ctx(ctx).Where("id", in.Id).Where("deleted_at IS NULL").Scan(&m); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if m == nil {
		return nil, xerror.New(xerror.CodeNotFound)
	}
	items, derr := decorateMessages(ctx, []*model.BizMessage{m}, 0)
	if derr != nil {
		return nil, derr
	}
	if len(items) == 0 {
		return nil, xerror.New(xerror.CodeNotFound)
	}
	return &v1.MessageDetailRes{MessageItem: items[0]}, nil
}

// SystemCreate 发布系统通知。事务: 写主表 + 批量写定向目标。
func (s *sMessage) SystemCreate(ctx context.Context, in *v1.MessageSystemCreateReq) (res *v1.MessageSystemCreateRes, err error) {
	if in.Level == 0 {
		in.Level = 1
	}
	if in.Status == 0 {
		in.Status = 1
	}
	// 范围=2/3 时必须传目标
	if in.TargetScope == model.MessageScopeRole || in.TargetScope == model.MessageScopeUser {
		if len(in.TargetIds) == 0 {
			return nil, xerror.New(xerror.CodeParamInvalid, "请选择目标角色或用户")
		}
	}
	senderId := contextx.UserId(ctx)
	var newId uint64
	err = dao.BizMessage.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		id, ie := tx.Model("biz_message").Ctx(ctx).Data(g.Map{
			"type":         model.MessageTypeSystem,
			"title":        in.Title,
			"content":      in.Content,
			"level":        in.Level,
			"status":       in.Status,
			"sender_id":    senderId,
			"target_scope": in.TargetScope,
			"receiver_id":  0,
		}).InsertAndGetId()
		if ie != nil {
			return ie
		}
		newId = uint64(id)
		if in.TargetScope == model.MessageScopeRole || in.TargetScope == model.MessageScopeUser {
			targetType := model.MessageTargetTypeRole
			if in.TargetScope == model.MessageScopeUser {
				targetType = model.MessageTargetTypeUser
			}
			rows := make([]g.Map, 0, len(in.TargetIds))
			for _, tid := range in.TargetIds {
				if tid == 0 {
					continue
				}
				rows = append(rows, g.Map{
					"message_id":  newId,
					"target_type": targetType,
					"target_id":   tid,
				})
			}
			if len(rows) > 0 {
				if _, ie = tx.Model("biz_message_target").Ctx(ctx).Data(rows).Insert(); ie != nil {
					return ie
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.MessageSystemCreateRes{Id: newId}, nil
}

// PrivateCreate 发送私信。
func (s *sMessage) PrivateCreate(ctx context.Context, in *v1.MessagePrivateCreateReq) (res *v1.MessagePrivateCreateRes, err error) {
	if in.Level == 0 {
		in.Level = 1
	}
	senderId := contextx.UserId(ctx)
	if senderId == 0 {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	if in.ReceiverId == senderId {
		return nil, xerror.New(xerror.CodeParamInvalid, "不能给自己发送私信")
	}
	// 校验接收人是否存在
	cnt, _ := dao.SysUser.Ctx(ctx).Where("id", in.ReceiverId).Where("deleted_at IS NULL").Count()
	if cnt == 0 {
		return nil, xerror.New(xerror.CodeUserNotFound, "接收人不存在")
	}
	id, err := dao.BizMessage.Ctx(ctx).Data(g.Map{
		"type":         model.MessageTypePrivate,
		"title":        in.Title,
		"content":      in.Content,
		"level":        in.Level,
		"status":       1,
		"sender_id":    senderId,
		"target_scope": model.MessageScopeNone,
		"receiver_id":  in.ReceiverId,
	}).InsertAndGetId()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.MessagePrivateCreateRes{Id: uint64(id)}, nil
}

// Delete 管理员软删除消息 (同时清理 target/read 关系)。
func (s *sMessage) Delete(ctx context.Context, in *v1.MessageDeleteReq) (res *v1.MessageDeleteRes, err error) {
	err = dao.BizMessage.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, ie := tx.Exec("UPDATE biz_message SET deleted_at=NOW() WHERE id=?", in.Id); ie != nil {
			return ie
		}
		if _, ie := tx.Model("biz_message_target").Ctx(ctx).Where("message_id", in.Id).Delete(); ie != nil {
			return ie
		}
		if _, ie := tx.Model("biz_message_read").Ctx(ctx).Where("message_id", in.Id).Delete(); ie != nil {
			return ie
		}
		return nil
	})
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.MessageDeleteRes{}, nil
}

// Inbox 我的收件箱分页。
func (s *sMessage) Inbox(ctx context.Context, in *v1.MessageInboxReq) (res *v1.MessageInboxRes, err error) {
	userId := contextx.UserId(ctx)
	if userId == 0 {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	roleIds, err := getUserRoleIds(ctx)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	q, err := buildInboxQuery(ctx, userId, roleIds, in.Type, in.Keyword, in.IsRead)
	if err != nil {
		return nil, err
	}
	total, err := q.Clone().Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	type scanRow struct {
		Id          uint64      `orm:"id"`
		Type        int         `orm:"type"`
		Title       string      `orm:"title"`
		Content     string      `orm:"content"`
		Level       int         `orm:"level"`
		Status      int         `orm:"status"`
		SenderId    uint64      `orm:"sender_id"`
		TargetScope int         `orm:"target_scope"`
		ReceiverId  uint64      `orm:"receiver_id"`
		CreatedAt   *gtime.Time `orm:"created_at"`
		UpdatedAt   *gtime.Time `orm:"updated_at"`
		IsRead      int         `orm:"is_read"`
	}
	var rows []*scanRow
	if err = q.Fields("m.id, m.type, m.title, m.content, m.level, m.status, m.sender_id, m.target_scope, m.receiver_id, m.created_at, m.updated_at, IF(r.id IS NULL, 0, 1) AS is_read").
		Order("m.id DESC").
		Page(in.Page, in.PageSize).
		Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	// 转为 BizMessage 列表用于装饰
	bms := make([]*model.BizMessage, 0, len(rows))
	readMap := make(map[uint64]bool, len(rows))
	for _, r := range rows {
		bms = append(bms, &model.BizMessage{
			Id: r.Id, Type: r.Type, Title: r.Title, Content: r.Content,
			Level: r.Level, Status: r.Status, SenderId: r.SenderId,
			TargetScope: r.TargetScope, ReceiverId: r.ReceiverId,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		})
		readMap[r.Id] = r.IsRead == 1
	}
	items, err := decorateMessages(ctx, bms, userId)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if v, ok := readMap[it.Id]; ok {
			it.IsRead = v
		}
	}
	page := response.Page(items, int64(total), in.Page, in.PageSize)
	r := v1.MessageInboxRes(page)
	return &r, nil
}

// InboxRead 阅读详情, 自动标记已读。
func (s *sMessage) InboxRead(ctx context.Context, in *v1.MessageInboxDetailReq) (res *v1.MessageInboxDetailRes, err error) {
	userId := contextx.UserId(ctx)
	if userId == 0 {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	roleIds, err := getUserRoleIds(ctx)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if !canAccess(ctx, in.Id, userId, roleIds) {
		return nil, xerror.New(xerror.CodeNotFound)
	}
	if err = upsertRead(ctx, in.Id, userId); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var m *model.BizMessage
	if err = dao.BizMessage.Ctx(ctx).Where("id", in.Id).Where("deleted_at IS NULL").Scan(&m); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if m == nil {
		return nil, xerror.New(xerror.CodeNotFound)
	}
	items, derr := decorateMessages(ctx, []*model.BizMessage{m}, userId)
	if derr != nil {
		return nil, derr
	}
	if len(items) == 0 {
		return nil, xerror.New(xerror.CodeNotFound)
	}
	items[0].IsRead = true
	return &v1.MessageInboxDetailRes{MessageItem: items[0]}, nil
}

// MarkRead 仅标记已读, 不返回详情。
func (s *sMessage) MarkRead(ctx context.Context, in *v1.MessageReadReq) (res *v1.MessageReadRes, err error) {
	userId := contextx.UserId(ctx)
	if userId == 0 {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	roleIds, err := getUserRoleIds(ctx)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if !canAccess(ctx, in.Id, userId, roleIds) {
		return nil, xerror.New(xerror.CodeNotFound)
	}
	if err = upsertRead(ctx, in.Id, userId); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	return &v1.MessageReadRes{}, nil
}

// MarkReadAll 全部标记为已读 (返回受影响条数)。
func (s *sMessage) MarkReadAll(ctx context.Context, in *v1.MessageReadAllReq) (res *v1.MessageReadAllRes, err error) {
	userId := contextx.UserId(ctx)
	if userId == 0 {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	roleIds, err := getUserRoleIds(ctx)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	q, err := buildInboxQuery(ctx, userId, roleIds, nil, "", intPtr(0))
	if err != nil {
		return nil, err
	}
	var ids []uint64
	if err = q.Fields("m.id").Scan(&ids); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if len(ids) == 0 {
		return &v1.MessageReadAllRes{Affected: 0}, nil
	}
	rows := make([]g.Map, 0, len(ids))
	for _, mid := range ids {
		rows = append(rows, g.Map{
			"message_id": mid,
			"user_id":    userId,
		})
	}
	if _, err = dao.BizMessageRead.Ctx(ctx).
		OnConflict("message_id", "user_id").
		Save(rows); err != nil {
		// 退化为逐条 upsert
		for _, mid := range ids {
			_ = upsertRead(ctx, mid, userId)
		}
	}
	return &v1.MessageReadAllRes{Affected: int64(len(ids))}, nil
}

// InboxDelete 个人删除 (从我的收件箱视角隐藏)。
func (s *sMessage) InboxDelete(ctx context.Context, in *v1.MessageInboxDeleteReq) (res *v1.MessageInboxDeleteRes, err error) {
	userId := contextx.UserId(ctx)
	if userId == 0 {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	roleIds, err := getUserRoleIds(ctx)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	if !canAccess(ctx, in.Id, userId, roleIds) {
		return nil, xerror.New(xerror.CodeNotFound)
	}
	// upsert hidden=1
	cnt, _ := dao.BizMessageRead.Ctx(ctx).
		Where("message_id", in.Id).Where("user_id", userId).Ctx(ctx).Count()
	if cnt > 0 {
		if _, err = dao.BizMessageRead.Ctx(ctx).
			Where("message_id", in.Id).Where("user_id", userId).
			Data(g.Map{"hidden": 1}).Update(); err != nil {
			return nil, xerror.Wrap(xerror.CodeBusinessError, err)
		}
	} else {
		if _, err = dao.BizMessageRead.Ctx(ctx).Data(g.Map{
			"message_id": in.Id,
			"user_id":    userId,
			"hidden":     1,
		}).Insert(); err != nil {
			return nil, xerror.Wrap(xerror.CodeBusinessError, err)
		}
	}
	return &v1.MessageInboxDeleteRes{}, nil
}

// UnreadCount 未读消息数量 (总数 / 系统通知 / 私信)。
func (s *sMessage) UnreadCount(ctx context.Context, in *v1.MessageUnreadCountReq) (res *v1.MessageUnreadCountRes, err error) {
	userId := contextx.UserId(ctx)
	if userId == 0 {
		return nil, xerror.New(xerror.CodeUnauthorized)
	}
	roleIds, err := getUserRoleIds(ctx)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	zero := 0
	q, err := buildInboxQuery(ctx, userId, roleIds, nil, "", &zero)
	if err != nil {
		return nil, err
	}
	type cnt struct {
		Type  int   `orm:"type"`
		Total int64 `orm:"total"`
	}
	var rows []*cnt
	if err = q.Fields("m.type AS type, COUNT(*) AS total").Group("m.type").Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	res = &v1.MessageUnreadCountRes{}
	for _, r := range rows {
		switch r.Type {
		case model.MessageTypeSystem:
			res.System = r.Total
		case model.MessageTypePrivate:
			res.Private = r.Total
		}
		res.Total += r.Total
	}
	return res, nil
}

// =====================================================================
// 私有辅助函数
// =====================================================================

// intPtr 工具: 取整型字面量地址。
func intPtr(v int) *int { return &v }

// buildInboxQuery 构造收件箱可见消息的基础查询 (带权限路由 + 隐藏过滤)。
//   - typeFilter: 1=系统, 2=私信; nil=全部
//   - keyword: 标题模糊
//   - isRead:  0=未读, 1=已读, nil=不过滤
func buildInboxQuery(
	ctx context.Context,
	userId uint64,
	roleIds []uint64,
	typeFilter *int,
	keyword string,
	isRead *int,
) (*gdb.Model, error) {
	q := dao.BizMessage.Ctx(ctx).As("m")
	// 安全说明: gf 的 LeftJoin ON 子句不支持占位符参数; 此处 userId 是 JWT claims 解出的
	// uint64, %d 格式化后不可能引入任意字符, 无注入面。禁止将此写法复制到字符串来源的场景。
	q = q.LeftJoin("biz_message_read r", fmt.Sprintf("r.message_id = m.id AND r.user_id = %d", userId))
	q = q.Where("m.deleted_at IS NULL").Where("m.status", 1)
	q = q.Where("IFNULL(r.hidden, 0) = 0")

	// 权限路由: 系统通知 + 私信
	args := []interface{}{}
	sysParts := []string{"m.target_scope = ?"}
	args = append(args, model.MessageScopeAll)
	if len(roleIds) > 0 {
		placeholders := strings.Repeat("?,", len(roleIds))
		placeholders = strings.TrimRight(placeholders, ",")
		sysParts = append(sysParts,
			"(m.target_scope = ? AND EXISTS(SELECT 1 FROM biz_message_target t WHERE t.message_id=m.id AND t.target_type=? AND t.target_id IN ("+placeholders+")))",
		)
		args = append(args, model.MessageScopeRole, model.MessageTargetTypeRole)
		for _, rid := range roleIds {
			args = append(args, rid)
		}
	}
	sysParts = append(sysParts,
		"(m.target_scope = ? AND EXISTS(SELECT 1 FROM biz_message_target t WHERE t.message_id=m.id AND t.target_type=? AND t.target_id=?))",
	)
	args = append(args, model.MessageScopeUser, model.MessageTargetTypeUser, userId)

	sysCond := "(m.type = ? AND (" + strings.Join(sysParts, " OR ") + "))"
	finalArgs := append([]interface{}{model.MessageTypeSystem}, args...)
	finalArgs = append(finalArgs, model.MessageTypePrivate, userId)
	privCond := "(m.type = ? AND m.receiver_id = ?)"

	q = q.Where(sysCond+" OR "+privCond, finalArgs...)

	if typeFilter != nil && (*typeFilter == model.MessageTypeSystem || *typeFilter == model.MessageTypePrivate) {
		q = q.Where("m.type", *typeFilter)
	}
	if keyword != "" {
		q = q.Where("m.title LIKE ?", "%"+strings.TrimSpace(keyword)+"%")
	}
	if isRead != nil {
		if *isRead == 1 {
			q = q.Where("r.id IS NOT NULL AND IFNULL(r.read_at, NULL) IS NOT NULL")
		} else if *isRead == 0 {
			q = q.Where("r.id IS NULL OR r.read_at IS NULL")
		}
	}
	return q, nil
}

// canAccess 校验当前用户是否有权访问该消息 (出现在收件箱中)。
func canAccess(ctx context.Context, msgId, userId uint64, roleIds []uint64) bool {
	q, err := buildInboxQuery(ctx, userId, roleIds, nil, "", nil)
	if err != nil {
		return false
	}
	cnt, err := q.Where("m.id", msgId).Count()
	if err != nil {
		return false
	}
	return cnt > 0
}

// upsertRead 标记单条消息已读 (按 unique key 幂等)。
func upsertRead(ctx context.Context, msgId, userId uint64) error {
	cnt, _ := dao.BizMessageRead.Ctx(ctx).
		Where("message_id", msgId).Where("user_id", userId).Ctx(ctx).Count()
	if cnt > 0 {
		_, err := dao.BizMessageRead.Ctx(ctx).
			Where("message_id", msgId).Where("user_id", userId).
			Data(g.Map{"read_at": gtime.Now()}).Update()
		return err
	}
	_, err := dao.BizMessageRead.Ctx(ctx).Data(g.Map{
		"message_id": msgId,
		"user_id":    userId,
		"read_at":    gtime.Now(),
		"hidden":     0,
	}).Insert()
	return err
}

// getUserRoleIds 通过 Casbin g 策略反查当前用户的角色 ID 列表。
func getUserRoleIds(ctx context.Context) ([]uint64, error) {
	u := contextx.LoginUser(ctx)
	if u == nil {
		return nil, nil
	}
	codes, err := casbinx.GetUserRoles(ctx, u.Username)
	if err != nil {
		return nil, err
	}
	if len(codes) == 0 {
		return nil, nil
	}
	values, err := dao.SysRole.Ctx(ctx).
		WhereIn("code", codes).
		Where("deleted_at IS NULL").
		Fields("id").
		Ctx(ctx).Array()
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(values))
	for _, v := range values {
		if id := v.Uint64(); id > 0 {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// decorateMessages 给消息列表附加 sender/receiver 名称、定向目标、(可选) 已读状态。
func decorateMessages(ctx context.Context, msgs []*model.BizMessage, userId uint64) ([]*model.MessageItem, error) {
	items := make([]*model.MessageItem, 0, len(msgs))
	if len(msgs) == 0 {
		return items, nil
	}
	// 收集相关用户/消息ID
	userIds := map[uint64]struct{}{}
	msgIds := make([]uint64, 0, len(msgs))
	for _, m := range msgs {
		msgIds = append(msgIds, m.Id)
		if m.SenderId > 0 {
			userIds[m.SenderId] = struct{}{}
		}
		if m.ReceiverId > 0 {
			userIds[m.ReceiverId] = struct{}{}
		}
	}
	// 批量查询用户名称
	userNameMap := map[uint64]string{}
	if len(userIds) > 0 {
		ids := make([]uint64, 0, len(userIds))
		for id := range userIds {
			ids = append(ids, id)
		}
		type uRow struct {
			Id       uint64 `orm:"id"`
			Username string `orm:"username"`
			Nickname string `orm:"nickname"`
		}
		var us []*uRow
		if err := dao.SysUser.Ctx(ctx).WhereIn("id", ids).Fields("id, username, nickname").
			Ctx(ctx).Scan(&us); err != nil {
			return nil, xerror.Wrap(xerror.CodeBusinessError, err)
		}
		for _, u := range us {
			name := u.Nickname
			if name == "" {
				name = u.Username
			}
			userNameMap[u.Id] = name
		}
	}
	// 批量查询定向目标
	type tRow struct {
		MessageId  uint64 `orm:"message_id"`
		TargetType int    `orm:"target_type"`
		TargetId   uint64 `orm:"target_id"`
	}
	targetMap := map[uint64]*struct {
		Roles []uint64
		Users []uint64
	}{}
	var ts []*tRow
	if err := dao.BizMessageTarget.Ctx(ctx).WhereIn("message_id", msgIds).
		Ctx(ctx).Scan(&ts); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	for _, t := range ts {
		entry, ok := targetMap[t.MessageId]
		if !ok {
			entry = &struct {
				Roles []uint64
				Users []uint64
			}{}
			targetMap[t.MessageId] = entry
		}
		if t.TargetType == model.MessageTargetTypeRole {
			entry.Roles = append(entry.Roles, t.TargetId)
		} else if t.TargetType == model.MessageTargetTypeUser {
			entry.Users = append(entry.Users, t.TargetId)
		}
	}
	// 收集所有出现的角色ID, 批量查名称
	roleNameMap := map[uint64]string{}
	roleIdSet := map[uint64]struct{}{}
	for _, e := range targetMap {
		for _, rid := range e.Roles {
			roleIdSet[rid] = struct{}{}
		}
	}
	if len(roleIdSet) > 0 {
		ids := make([]uint64, 0, len(roleIdSet))
		for id := range roleIdSet {
			ids = append(ids, id)
		}
		type rRow struct {
			Id   uint64 `orm:"id"`
			Name string `orm:"name"`
		}
		var rs []*rRow
		if err := dao.SysRole.Ctx(ctx).WhereIn("id", ids).Fields("id, name").
			Ctx(ctx).Scan(&rs); err != nil {
			return nil, xerror.Wrap(xerror.CodeBusinessError, err)
		}
		for _, r := range rs {
			roleNameMap[r.Id] = r.Name
		}
	}
	// 批量查询当前用户的已读状态
	readMap := map[uint64]bool{}
	if userId > 0 {
		type rdRow struct {
			MessageId uint64 `orm:"message_id"`
		}
		var reads []*rdRow
		_ = dao.BizMessageRead.Ctx(ctx).
			WhereIn("message_id", msgIds).Where("user_id", userId).
			Where("read_at IS NOT NULL").
			Fields("message_id").Ctx(ctx).Scan(&reads)
		for _, r := range reads {
			readMap[r.MessageId] = true
		}
	}
	// 组装 VO
	for _, m := range msgs {
		item := &model.MessageItem{
			Id:           m.Id,
			Type:         m.Type,
			Title:        m.Title,
			Content:      m.Content,
			Level:        m.Level,
			Status:       m.Status,
			SenderId:     m.SenderId,
			SenderName:   userNameMap[m.SenderId],
			TargetScope:  m.TargetScope,
			ReceiverId:   m.ReceiverId,
			ReceiverName: userNameMap[m.ReceiverId],
			CreatedAt:    m.CreatedAt,
			UpdatedAt:    m.UpdatedAt,
		}
		if e, ok := targetMap[m.Id]; ok {
			item.TargetRoleIds = e.Roles
			item.TargetUserIds = e.Users
			names := make([]string, 0, len(e.Roles))
			for _, rid := range e.Roles {
				if n := roleNameMap[rid]; n != "" {
					names = append(names, n)
				}
			}
			item.TargetRoleNames = names
		}
		if userId > 0 {
			item.IsRead = readMap[m.Id]
		}
		items = append(items, item)
	}
	return items, nil
}

// Package online 在线会话管理: 登录注册 / 心跳续活跃 / 强制下线。
//
// 存储设计: 单个 Redis HASH (consts.OnlineSessionKey),
// field 为会话 ID (token 的 SHA-256 摘要前 16 位, 避免在 key/field 中存原始 token),
// value 为 model.OnlineSession 的 JSON 序列化。
// 会话过期不依赖 Redis TTL (HASH 不支持整体按 field 过期的兼容写法),
// 而是在列表查询时按 loginAt + jwt 有效期惰性清理。
package online

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/gogf/gf/v2/database/gredis"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/jwtx"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

type sOnline struct{}

func init() {
	service.RegisterOnline(NewOnline())
}

func NewOnline() *sOnline {
	return &sOnline{}
}

// SessionId 由 token 计算会话 ID: SHA-256 摘要前 16 个 hex 字符。
// 摘要不可逆, 泄露会话 ID 无法还原 token。
func SessionId(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])[:16]
}

// Register 登录成功后注册在线会话 (尽力而为: 失败仅记日志, 不影响登录)。
func (s *sOnline) Register(ctx context.Context, token string, sess model.OnlineSession) {
	if token == "" {
		return
	}
	now := time.Now().Unix()
	if sess.LoginAt == 0 {
		sess.LoginAt = now
	}
	sess.Token = token
	sess.LastActiveAt = now
	s.write(ctx, SessionId(token), &sess)
}

// Touch 刷新会话活跃时间。
// 节流: 每个会话 consts.OnlineTouchIntervalSec 内只真正写一次,
// 由 Auth 中间件异步调用, 任何失败静默忽略 (心跳属于遥测数据, 不允许影响请求)。
func (s *sOnline) Touch(ctx context.Context, token string) {
	if token == "" {
		return
	}
	sid := SessionId(token)
	touchKey := consts.OnlineTouchPrefix + sid
	// SET NX EX: 未命中说明节流窗口内已刷新过, 直接跳过
	set, err := g.Redis().GroupString().Set(ctx, touchKey, 1, gredis.SetOption{
		NX:        true,
		TTLOption: gredis.TTLOption{EX: ptrInt64(consts.OnlineTouchIntervalSec)},
	})
	if err != nil || set == nil || set.IsNil() {
		return
	}
	raw, err := g.Redis().GroupHash().HGet(ctx, consts.OnlineSessionKey, sid)
	if err != nil || raw == nil || raw.IsNil() {
		return
	}
	var sess model.OnlineSession
	if err = json.Unmarshal([]byte(raw.String()), &sess); err != nil {
		return
	}
	sess.LastActiveAt = time.Now().Unix()
	s.write(ctx, sid, &sess)
}

// Remove 会话登出时剔除 (黑名单由调用方负责)。
func (s *sOnline) Remove(ctx context.Context, token string) {
	if token == "" {
		return
	}
	_, _ = g.Redis().GroupHash().HDel(ctx, consts.OnlineSessionKey, SessionId(token))
}

// List 在线用户分页列表 (按最近活跃倒序), 惰性清理已过期会话。
func (s *sOnline) List(ctx context.Context, req *v1.OnlineListReq) (res *v1.OnlineListRes, err error) {
	all, err := g.Redis().GroupHash().HGetAll(ctx, consts.OnlineSessionKey)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "读取在线会话失败")
	}
	now := time.Now().Unix()
	expireSec := jwtx.ExpireSec(ctx)

	var (
		items   []*model.OnlineUserItem
		expired []string
	)
	for field, v := range all.MapStrStr() {
		var sess model.OnlineSession
		if err = json.Unmarshal([]byte(v), &sess); err != nil {
			continue
		}
		// 惰性清理: loginAt 已超过 token 有效期的会话视为过期
		if sess.LoginAt+expireSec < now {
			expired = append(expired, field)
			continue
		}
		if req.Username != "" && !strings.Contains(sess.Username, req.Username) {
			continue
		}
		items = append(items, &model.OnlineUserItem{
			SessionId:    field,
			UserId:       sess.UserId,
			Username:     sess.Username,
			Nickname:     sess.Nickname,
			Ip:           sess.Ip,
			UserAgent:    sess.UserAgent,
			LoginAt:      gtime.NewFromTimeStamp(sess.LoginAt),
			LastActiveAt: gtime.NewFromTimeStamp(sess.LastActiveAt),
		})
	}
	if len(expired) > 0 {
		_, _ = g.Redis().GroupHash().HDel(ctx, consts.OnlineSessionKey, expired...)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].LastActiveAt.Timestamp() > items[j].LastActiveAt.Timestamp()
	})

	// 内存分页
	page, pageSize := req.Page, req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	if start > len(items) {
		start = len(items)
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}

	total := int64(len(items))
	r := v1.OnlineListRes(response.Page(items[start:end], total, page, pageSize))
	return &r, nil
}

// Kick 强制下线: 会话 token 写入黑名单并从在线列表剔除。
func (s *sOnline) Kick(ctx context.Context, req *v1.OnlineKickReq) (res *v1.OnlineKickRes, err error) {
	raw, err := g.Redis().GroupHash().HGet(ctx, consts.OnlineSessionKey, req.Id)
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "读取会话失败")
	}
	if raw == nil || raw.IsNil() {
		return nil, xerror.New(xerror.CodeBusinessError, "会话不存在或已下线")
	}
	var sess model.OnlineSession
	if err = json.Unmarshal([]byte(raw.String()), &sess); err != nil || sess.Token == "" {
		return nil, xerror.New(xerror.CodeBusinessError, "会话数据异常")
	}
	if err = jwtx.Blacklist(ctx, sess.Token); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "下线失败")
	}
	_, _ = g.Redis().GroupHash().HDel(ctx, consts.OnlineSessionKey, req.Id)
	return &v1.OnlineKickRes{}, nil
}

// write 写入单条会话记录 (尽力而为)。
func (s *sOnline) write(ctx context.Context, sid string, sess *model.OnlineSession) {
	data, err := json.Marshal(sess)
	if err != nil {
		return
	}
	if _, err = g.Redis().GroupHash().HSet(ctx, consts.OnlineSessionKey, g.Map{sid: string(data)}); err != nil {
		g.Log().Errorf(ctx, "写入在线会话失败: %v", err)
	}
}

// ptrInt64 返回 int64 指针 (Redis SetOption 的 EX 字段需要)。
func ptrInt64(v int64) *int64 {
	return &v
}

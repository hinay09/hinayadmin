// Package notify 站内消息实时推送枢纽 (进程内 SSE + Redis 跨实例广播)。
//
// 触达模型:
//   - 本实例: 每个在线用户的 SSE 连接注册一个 channel, 消息按接收人投递,
//     连接端收到后刷新未读数并弹出提示;
//   - 跨实例: Publish* 双写 —— 本实例直投 + 发布到 Redis 频道
//     hinay:notify:bus:{tenantId} (多租户 §9 缓存键规约: 频道带租户维度);
//     各实例惰性订阅 (首个 SSE 连接建立时启动), 收到后回放到本地 hub。
//     接收人连接不在发布实例上也能实时触达; Redis 不可用时退化为单实例行为
//     (本地直投不受影响, 未触达的由前端轮询兜底)。
//
// 资源保护: 每用户连接数上限 MaxConnsPerUser, 全局上限 MaxConnsTotal,
// 超限订阅返回错误 (连接端应答 429), 防止连接数无限增长耗尽内存。
package notify

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gogf/gf/v2/database/gredis"
	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/utility/contextx"
)

// 连接数上限 (资源保护)。
const (
	// MaxConnsPerUser 单用户并发 SSE 连接上限 (多标签页/多设备正常场景 3 条足够,
	// 超出通常是连接泄漏或异常客户端)。
	MaxConnsPerUser = 3
	// MaxConnsTotal 全实例连接总上限 (内存兜底, 与用户数无关)。
	MaxConnsTotal = 1000
)

// Event 推送给单个连接的事件。
type Event struct {
	Type  string `json:"type"`  // message=新消息
	Title string `json:"title"` // 消息标题
	Level int    `json:"level"` // 消息级别
}

// BusEvent Redis 广播消息体 (跨实例传输; TenantId 仅用于诊断, 频道本身已按租户隔离;
// Origin 为发布实例标识, 订阅端借此跳过自己发布的消息 —— 本地已直投过, 回放会重复)。
type BusEvent struct {
	TenantId uint64   `json:"tenantId"`
	Origin   string   `json:"origin"`
	All      bool     `json:"all,omitempty"`
	UserIds  []uint64 `json:"userIds,omitempty"`
	Title    string   `json:"title"`
	Level    int      `json:"level"`
}

// instanceID 本实例标识 (进程启动随机生成): 广播自过滤用。
var instanceID = randomInstanceID()

// randomInstanceID 8 字节随机 hex。
func randomInstanceID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("p%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// hub 连接注册表 (按用户 ID 分组)。
type hub struct {
	mu    sync.RWMutex
	subs  map[uint64]map[chan Event]struct{}
	total int // 当前连接总数 (与 subs 一致性由 mu 保证)
}

var h = &hub{subs: make(map[uint64]map[chan Event]struct{})}

// ---------------------------------------------------------------------------
// 本地投递
// ---------------------------------------------------------------------------

// localDeliver 向本地 hub 的指定用户投递 (直投与订阅回放共用), 去重后逐用户分发。
func localDeliver(userIds []uint64, title string, level int) {
	for _, id := range dedupeIds(userIds) {
		h.mu.RLock()
		for c := range h.subs[id] {
			select {
			case c <- Event{Type: "message", Title: title, Level: level}:
			default: // 慢消费者丢事件, 未读数由轮询对齐
			}
		}
		h.mu.RUnlock()
	}
}

// localDeliverAll 向本地全部连接投递。
func localDeliverAll(title string, level int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, set := range h.subs {
		for c := range set {
			select {
			case c <- Event{Type: "message", Title: title, Level: level}:
			default:
			}
		}
	}
}

// ---------------------------------------------------------------------------
// 发布 (双写: 本地直投 + Redis 广播)
// ---------------------------------------------------------------------------

// tenantIdOf 广播频道租户维度: 无租户上下文按平台租户 (单租户部署即平台)。
func tenantIdOf(ctx context.Context) uint64 {
	if tid := contextx.TenantId(ctx); tid > 0 {
		return tid
	}
	return consts.TenantPlatformId
}

// PublishTo 向指定用户发布事件: 本地连接直投 + 广播到其他实例 (含本实例订阅回放)。
// Redis 发布失败仅记日志 (尽力而为, 本地直投不受影响)。
func PublishTo(ctx context.Context, userIds []uint64, title string, level int) {
	if len(userIds) == 0 {
		return
	}
	localDeliver(userIds, title, level)
	publishBus(ctx, &BusEvent{TenantId: tenantIdOf(ctx), UserIds: dedupeIds(userIds), Title: title, Level: level})
}

// PublishAll 向全部用户发布 (全员系统通知)。
func PublishAll(ctx context.Context, title string, level int) {
	localDeliverAll(title, level)
	publishBus(ctx, &BusEvent{TenantId: tenantIdOf(ctx), All: true, Title: title, Level: level})
}

// publishBus 发布到本租户频道 (尽力而为)。
func publishBus(ctx context.Context, ev *BusEvent) {
	ev.Origin = instanceID
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	channel := consts.NotifyBusPrefix + strconv.FormatUint(ev.TenantId, 10)
	if _, err = g.Redis().GroupPubSub().Publish(ctx, channel, payload); err != nil {
		g.Log().Debugf(ctx, "notify bus publish failed (本地直投不受影响): %v", err)
	}
}

// dedupeIds 用户ID去重去零 (投递与广播体共用)。
func dedupeIds(ids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(ids))
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// ---------------------------------------------------------------------------
// 订阅 (惰性启动: 首个本地 SSE 连接建立时开始, 无连接的实例不占用订阅)
// ---------------------------------------------------------------------------

var busOnce sync.Once

// startSubscriber 启动 Redis 模式订阅 (进程生命周期, 断线指数退避重连)。
func startSubscriber() {
	busOnce.Do(func() {
		go subscriberLoop()
	})
}

// subscriberLoop 订阅 hinay:notify:bus:* 并回放到本地 hub; 断开按指数退避重连 (上限 30s)。
func subscriberLoop() {
	ctx := context.Background()
	backoff := time.Second
	for {
		err := subscribeOnce(ctx)
		if err == nil {
			return // 进程退出
		}
		g.Log().Warningf(ctx, "notify bus 订阅断开, %s 后重连: %v", backoff, err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
	}
}

// subscribeOnce 建立一次模式订阅并循环收消息, 返回时连接已关闭。
func subscribeOnce(ctx context.Context) error {
	conn, _, err := g.Redis().GroupPubSub().PSubscribe(ctx, consts.NotifyBusPattern)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	for {
		v, rerr := conn.Receive(ctx)
		if rerr != nil {
			return rerr
		}
		if v == nil {
			continue
		}
		msg, ok := v.Val().(*gredis.Message)
		if !ok || msg == nil {
			continue // 订阅确认帧等非业务消息
		}
		var ev BusEvent
		if jerr := json.Unmarshal([]byte(msg.Payload), &ev); jerr != nil {
			g.Log().Debugf(ctx, "notify bus 忽略无法解析的消息: %v", jerr)
			continue
		}
		if ev.Origin == instanceID {
			continue // 自己发布的消息本地已直投, 回放会重复
		}
		if ev.All {
			localDeliverAll(ev.Title, ev.Level)
		} else {
			localDeliver(ev.UserIds, ev.Title, ev.Level)
		}
	}
}

// Subscribe 订阅当前用户的事件流; cancel 必须在连接结束时调用(幂等), 否则占用连接配额。
// 超出 MaxConnsPerUser / MaxConnsTotal 时返回错误 (连接端应答 429)。
func Subscribe(userId uint64) (ch chan Event, cancel func(), err error) {
	startSubscriber() // 惰性启动跨实例订阅
	if userId == 0 {
		c := make(chan Event, 8)
		return c, func() {}, nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.total >= MaxConnsTotal {
		return nil, nil, fmt.Errorf("连接总数已达上限(%d), 请稍后重试", MaxConnsTotal)
	}
	if len(h.subs[userId]) >= MaxConnsPerUser {
		return nil, nil, fmt.Errorf("同一账号的连接数已达上限(%d), 请关闭多余页面后重试", MaxConnsPerUser)
	}
	c := make(chan Event, 8)
	if h.subs[userId] == nil {
		h.subs[userId] = make(map[chan Event]struct{})
	}
	h.subs[userId][c] = struct{}{}
	h.total++
	return c, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if set := h.subs[userId]; set != nil {
			if _, ok := set[c]; ok { // 幂等: 重复 cancel 不重复扣减
				delete(set, c)
				h.total--
			}
			if len(set) == 0 {
				delete(h.subs, userId)
			}
		}
	}, nil
}

// Online 是否有任一在线订阅 (可用于调试/监控)。
func Online() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	n := 0
	for _, set := range h.subs {
		n += len(set)
	}
	return n
}

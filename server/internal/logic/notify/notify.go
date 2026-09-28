// Package notify 站内消息实时推送枢纽 (进程内, SSE)。
//
// 单实例内存态: 每个在线用户的 SSE 连接注册一个 channel,
// 消息创建方按接收人发布事件, 连接端收到后刷新未读数并弹出提示。
// 多副本部署时各实例独立 (消息只在接收人连接的实例上触达,
// 未触达的由前端轮询兜底), 单实例部署无此问题。
package notify

import (
	"sync"
)

// Event 推送给单个连接的事件。
type Event struct {
	Type  string `json:"type"`  // message=新消息
	Title string `json:"title"` // 消息标题
	Level int    `json:"level"` // 消息级别
}

// hub 连接注册表 (按用户 ID 分组)。
type hub struct {
	mu   sync.RWMutex
	subs map[uint64]map[chan Event]struct{}
}

var h = &hub{subs: make(map[uint64]map[chan Event]struct{})}

// Subscribe 订阅当前用户的事件流; cancel 必须在连接结束时调用, 否则 channel 泄漏。
func Subscribe(userId uint64) (ch chan Event, cancel func()) {
	if userId == 0 {
		c := make(chan Event, 8)
		return c, func() {}
	}
	c := make(chan Event, 8)
	h.mu.Lock()
	if h.subs[userId] == nil {
		h.subs[userId] = make(map[chan Event]struct{})
	}
	h.subs[userId][c] = struct{}{}
	h.mu.Unlock()
	return c, func() {
		h.mu.Lock()
		if set := h.subs[userId]; set != nil {
			delete(set, c)
			if len(set) == 0 {
				delete(h.subs, userId)
			}
		}
		h.mu.Unlock()
	}
}

// PublishTo 向指定用户的所有在线连接发布事件 (无在线连接则丢弃, 由轮询兜底)。
func PublishTo(userIds []uint64, title string, level int) {
	if len(userIds) == 0 {
		return
	}
	seen := make(map[uint64]struct{}, len(userIds))
	for _, id := range userIds {
		if id == 0 {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
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

// PublishAll 向全部在线连接发布 (全员系统通知)。
func PublishAll(title string, level int) {
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

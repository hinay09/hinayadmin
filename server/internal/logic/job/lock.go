// Package job 分布式锁: Redis SET NX PX + 属主令牌 + 看门狗续期。
//
// 语义 (sys_job.concurrent_policy = "skip" 时生效):
//   - 触发时抢占 hinay:job:lock:{tenantId}:{jobId}, 抢到的实例执行, 其余实例跳过本次触发;
//   - 执行期间看门狗按 TTL/3 周期校验属主并续期, 任务结束即释放 (Lua 比较属主后删除,
//     不会误删他人接续的新锁);
//   - 实例崩溃后锁最多存活一个 TTL, 由其他实例自然接管 —— 无需人工干预。
//
// 降级: Redis 不可用/未配置时 fail-open 照常执行 (退化为单实例行为, 仅记警告) ——
// 分布式锁是防重复执行的优化, 不是正确性边界 (任务自身仍应尽量幂等)。
package job

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"hinay.cn/admin/internal/consts"
)

// lockKey 完整锁键: 租户维度 + 任务ID (多租户 §9 缓存键规约)。
func lockKey(tenantId, jobId uint64) string {
	return fmt.Sprintf("%s%d:%d", consts.JobLockPrefix, tenantId, jobId)
}

// luaRelease 仅当值仍为本实例令牌时删除 (防误删他人锁)。
const luaRelease = `if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) else return 0 end`

// luaRenew 仅当值仍为本实例令牌时续期。
const luaRenew = `if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("PEXPIRE", KEYS[1], ARGV[2]) else return 0 end`

// tryLock 尝试抢占分布式锁。
// 返回 ok=false 表示锁被其他实例持有, 调用方应跳过本次触发;
// ok=true 时返回 release (释放并停止看门狗), 必须在执行结束后调用。
// Redis 不可用时 fail-open: ok=true + 空操作 release (退化为单实例行为)。
func tryLock(ctx context.Context, tenantId, jobId uint64) (ok bool, release func()) {
	token := randomToken()
	key := lockKey(tenantId, jobId)
	ttlMs := int64(consts.JobLockTTLSec * 1000)

	// SET key token NX PX ttl: 原子抢占
	v, err := g.Redis().Do(ctx, "SET", key, token, "NX", "PX", ttlMs)
	if err != nil {
		// Redis 不可用: fail-open, 退化为无锁执行 (单实例语义)
		g.Log().Warningf(ctx, "分布式锁不可用(job %d), 无锁执行: %v", jobId, err)
		return true, func() {}
	}
	if v == nil || !v.Bool() {
		// 已被其他实例持有: 跳过本次触发 (赢者写执行日志, 此处静默)
		return false, nil
	}

	// 看门狗: 执行超过 TTL 时按 TTL/3 续期, 属主丢失即停 (最长存活一个 TTL 的接管语义)
	wctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Duration(consts.JobLockTTLSec) * time.Second / 3)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-wctx.Done():
				return
			case <-ticker.C:
				if _, err := g.Redis().Do(wctx, "EVAL", luaRenew, 1, key, token, ttlMs); err != nil {
					return // 续期失败放弃属主 (锁自然过期, 他人可接管)
				}
			}
		}
	}()

	return true, func() {
		cancel()
		close(stop)
		if _, err := g.Redis().Do(context.WithoutCancel(ctx), "EVAL", luaRelease, 1, key, token); err != nil {
			g.Log().Warningf(ctx, "释放任务锁失败(job %d, 锁将按 TTL 过期): %v", jobId, err)
		}
	}
}

// randomToken 16 字节随机属主令牌 (hex), 区分持锁实例。
func randomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand 失败极罕见, 退化为纳秒时间戳 (仍可区分同批触发)
		return fmt.Sprintf("t%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// normalizeConcurrentPolicy 入库前归一: 仅接受 allow/skip, 其余(含空)回落 allow。
func normalizeConcurrentPolicy(p string) string {
	if p == consts.JobConcurrentSkip {
		return consts.JobConcurrentSkip
	}
	return consts.JobConcurrentAllow
}

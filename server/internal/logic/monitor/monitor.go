// Package monitor 服务器监控: CPU/内存/磁盘/主机/Go 运行时指标 (gopsutil)
// 与 MySQL/Redis 连接状态, 运维排查的第一入口。
//
// 权限: 仅超管。除 Casbin 策略 (种子只给 admin 角色) 外, 每个方法入口
// 再做 contextx.IsAdmin 硬校验 —— 主机指标与中间件拓扑属于基础设施信息,
// 即使 Casbin 策略被误配也不应暴露给普通用户。
package monitor

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"

	v1 "hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/contextx"
	"hinay.cn/admin/utility/xerror"
)

// processStart 进程启动时间, 用于计算应用运行时长。
var processStart = time.Now()

type sMonitor struct{}

func init() {
	service.RegisterMonitor(NewMonitor())
}

func NewMonitor() *sMonitor {
	return &sMonitor{}
}

// guardAdmin 仅超管可见的服务器指标入口。
func guardAdmin(ctx context.Context) error {
	if !contextx.IsAdmin(ctx) {
		return xerror.New(xerror.CodeForbidden)
	}
	return nil
}

// skipFsTypes 虚拟/只读/容器挂载文件系统, 不属于"磁盘用量"监控范畴。
var skipFsTypes = map[string]bool{
	"proc": true, "sysfs": true, "devtmpfs": true, "tmpfs": true, "devfs": true,
	"squashfs": true, "iso9660": true, "overlay": true, "aufs": true,
	"nsfs": true, "cgroup": true, "cgroup2": true, "efivarfs": true,
	"fusectl": true, "mqueue": true, "hugetlbfs": true, "binfmt_misc": true,
}

// Server 服务器监控指标。
// 指标采集全部"尽力而为": 单项失败置零值不报错, 监控页不应因某个
// 子系统不可用 (如容器内无 /proc 权限) 整页不可用。
func (s *sMonitor) Server(ctx context.Context, _ *v1.MonitorServerReq) (res *v1.MonitorServerRes, err error) {
	if gerr := guardAdmin(ctx); gerr != nil {
		return nil, gerr
	}
	out := &v1.MonitorServerRes{}

	// 主机信息
	if hi, herr := host.Info(); herr == nil {
		out.Host = v1.MonitorHost{
			Hostname:      hi.Hostname,
			OS:            hi.OS,
			Platform:      strings.TrimSpace(hi.Platform + " " + hi.PlatformVersion),
			KernelArch:    hi.KernelArch,
			KernelVersion: hi.KernelVersion,
			BootTime:      int64(hi.BootTime),
			UptimeSec:     hi.Uptime,
		}
	}
	out.Host.AppUptimeSec = int64(time.Since(processStart).Seconds())
	out.Host.ServerTime = time.Now().Unix()

	// CPU: interval=0 取自上次调用以来的增量 (gopsutil 内部保存上次采样点),
	// 前端轮询间隔即采样窗口; 首次调用返回自开机以来的平均值。
	if per, cerr := cpu.Percent(0, true); cerr == nil {
		out.Cpu.PerCore = per
		total := 0.0
		for _, v := range per {
			total += v
		}
		if len(per) > 0 {
			out.Cpu.UsagePercent = total / float64(len(per))
		}
	}
	out.Cpu.CoresLogical, _ = cpu.Counts(true)
	out.Cpu.CoresPhysical, _ = cpu.Counts(false)
	if infos, ierr := cpu.Info(); ierr == nil && len(infos) > 0 {
		out.Cpu.ModelName = strings.TrimSpace(infos[0].ModelName)
		out.Cpu.Mhz = infos[0].Mhz
	}
	if avg, lerr := load.Avg(); lerr == nil {
		out.Cpu.Load1, out.Cpu.Load5, out.Cpu.Load15 = avg.Load1, avg.Load5, avg.Load15
	}

	// 内存
	if vm, merr := mem.VirtualMemory(); merr == nil {
		out.Memory = v1.MonitorMemory{
			Total:       vm.Total,
			Used:        vm.Used,
			Available:   vm.Available,
			UsedPercent: vm.UsedPercent,
		}
	}

	// 磁盘: 过滤虚拟文件系统后逐分区取用量
	if parts, derr := disk.Partitions(false); derr == nil {
		for _, p := range parts {
			if skipFsTypes[p.Fstype] {
				continue
			}
			// macOS: APFS 卷组内的合成卷 (/System/Volumes/VM、Preboot、Update 等)
			// 与 / 挂的是同一个容器, 重复统计同一份用量, 只保留 / 与真实用户挂载卷
			if strings.HasPrefix(p.Mountpoint, "/System/Volumes/") || p.Mountpoint == "/Volumes/Recovery" {
				continue
			}
			du, uerr := disk.Usage(p.Mountpoint)
			if uerr != nil || du.Total == 0 {
				continue
			}
			out.Disks = append(out.Disks, &v1.MonitorDisk{
				Device:      p.Device,
				Mount:       p.Mountpoint,
				Fstype:      p.Fstype,
				Total:       du.Total,
				Used:        du.Used,
				Free:        du.Free,
				UsedPercent: du.UsedPercent,
			})
		}
	}

	// Go 运行时
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	out.Go = v1.MonitorGo{
		GoVersion:      runtime.Version(),
		Goroutines:     runtime.NumGoroutine(),
		GoMaxProcs:     runtime.GOMAXPROCS(0),
		HeapAlloc:      ms.HeapAlloc,
		HeapSys:        ms.HeapSys,
		StackSys:       ms.StackSys,
		Sys:            ms.Sys,
		NumGC:          ms.NumGC,
		GCPauseTotalMs: float64(ms.PauseTotalNs) / 1e6,
		LastGCPauseMs:  float64(ms.PauseNs[(ms.NumGC+255)%256]) / 1e6,
	}

	return out, nil
}

// Mysql MySQL 运行状态: SHOW GLOBAL STATUS 全量取回后按需挑取,
// 避免逐条 LIKE 多次往返。
func (s *sMonitor) Mysql(ctx context.Context, _ *v1.MonitorMysqlReq) (res *v1.MonitorMysqlRes, err error) {
	if gerr := guardAdmin(ctx); gerr != nil {
		return nil, gerr
	}
	out := &v1.MonitorMysqlRes{}

	if v, verr := g.DB().GetValue(ctx, "SELECT VERSION()"); verr == nil {
		out.Version = v.String()
	}
	rows, serr := g.DB().GetAll(ctx, "SHOW GLOBAL STATUS")
	if serr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, serr, "查询 MySQL 状态失败")
	}
	st := make(map[string]string, len(rows))
	for _, r := range rows {
		st[r["Variable_name"].String()] = r["Value"].String()
	}
	out.UptimeSec = parseUint(st["Uptime"])
	out.ThreadsConnected = parseInt64(st["Threads_connected"])
	out.ThreadsRunning = parseInt64(st["Threads_running"])
	out.MaxUsedConnections = parseInt64(st["Max_used_connections"])
	out.AbortedConnects = parseUint(st["Aborted_connects"])
	out.SlowQueries = parseUint(st["Slow_queries"])
	out.Queries = parseUint(st["Questions"])
	out.BytesReceived = parseUint(st["Bytes_received"])
	out.BytesSent = parseUint(st["Bytes_sent"])

	// 应用侧连接池 (go-sql-driver): 与服务端 Threads_* 互补,
	// InUse 持续逼近 Open 即池打满信号
	for _, item := range g.DB().Stats(ctx) {
		node, stat := item.Node(), item.Stats()
		out.Pool = v1.MonitorDbPool{
			Host:           node.Host + ":" + node.Port,
			Name:           node.Name,
			Open:           stat.OpenConnections,
			InUse:          stat.InUse,
			Idle:           stat.Idle,
			WaitCount:      stat.WaitCount,
			WaitDurationMs: stat.WaitDuration.Milliseconds(),
		}
		break // 单节点部署取第一个; 未来多节点可扩展为数组
	}
	return out, nil
}

// Redis Redis 运行状态: INFO 文本解析 + DBSIZE。
func (s *sMonitor) Redis(ctx context.Context, _ *v1.MonitorRedisReq) (res *v1.MonitorRedisRes, err error) {
	if gerr := guardAdmin(ctx); gerr != nil {
		return nil, gerr
	}
	out := &v1.MonitorRedisRes{}

	info, ierr := g.Redis().Do(ctx, "INFO")
	if ierr != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, ierr, "查询 Redis 状态失败")
	}
	kv := parseRedisInfo(info.String())
	out.Version = kv["redis_version"]
	out.UptimeSec = parseInt64(kv["uptime_in_seconds"])
	out.ConnectedClients = parseInt64(kv["connected_clients"])
	out.UsedMemory = parseUint(kv["used_memory"])
	out.UsedMemoryHuman = kv["used_memory_human"]
	out.UsedMemoryPeakHuman = kv["used_memory_peak_human"]
	out.TotalConnections = parseUint(kv["total_connections_received"])
	out.OpsPerSec = parseInt64(kv["instantaneous_ops_per_sec"])
	out.KeyspaceHits = parseUint(kv["keyspace_hits"])
	out.KeyspaceMisses = parseUint(kv["keyspace_misses"])
	if hits, misses := out.KeyspaceHits, out.KeyspaceMisses; hits+misses > 0 {
		out.HitRatePercent = float64(hits) / float64(hits+misses) * 100
	} else {
		out.HitRatePercent = -1
	}
	if size, derr := g.Redis().Do(ctx, "DBSIZE"); derr == nil {
		out.DbSize = size.Int64()
	}
	return out, nil
}

// parseRedisInfo 解析 INFO 响应文本为键值 map (忽略注释与空行)。
func parseRedisInfo(text string) map[string]string {
	kv := make(map[string]string)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, ":"); ok {
			kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return kv
}

// parseUint 宽松解析无符号整数, 失败返回 0 (监控项尽力而为)。
func parseUint(s string) uint64 {
	v, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return v
}

// parseInt64 宽松解析整数, 失败返回 0。
func parseInt64(s string) int64 {
	v, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return v
}

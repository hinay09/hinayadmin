// Package v1 系统监控-服务器监控接口 (仅超管)。
package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

// MonitorServerReq 服务器指标 (CPU/内存/磁盘/主机信息/Go 运行时)。
type MonitorServerReq struct {
	g.Meta `path:"/system/monitor/server" tags:"SystemMonitor" method:"get" summary:"服务器监控指标(仅超管)"`
}

// MonitorServerRes 服务器指标响应。
// CPU 占用为自上次调用以来的增量值 (gopsutil 内部维护上次采样点),
// 前端轮询间隔即为采样窗口; 首次调用返回自开机以来的平均值。
type MonitorServerRes struct {
	Host   MonitorHost    `json:"host"   dc:"主机信息"`
	Cpu    MonitorCpu     `json:"cpu"    dc:"CPU 指标"`
	Memory MonitorMemory  `json:"memory" dc:"内存指标"`
	Disks  []*MonitorDisk `json:"disks"  dc:"磁盘分区(已过滤虚拟/只读文件系统)"`
	Go     MonitorGo      `json:"go"     dc:"Go 运行时指标"`
}

// MonitorHost 主机信息。
type MonitorHost struct {
	Hostname      string `json:"hostname"      dc:"主机名"`
	OS            string `json:"os"            dc:"操作系统 (darwin/linux/windows)"`
	Platform      string `json:"platform"      dc:"发行版名称及版本"`
	KernelArch    string `json:"kernelArch"    dc:"内核架构"`
	KernelVersion string `json:"kernelVersion" dc:"内核版本"`
	BootTime      int64  `json:"bootTime"      dc:"开机时间戳(秒)"`
	UptimeSec     uint64 `json:"uptimeSec"     dc:"已运行时长(秒)"`
	AppUptimeSec  int64  `json:"appUptimeSec"  dc:"应用已运行时长(秒)"`
	ServerTime    int64  `json:"serverTime"    dc:"服务器当前时间戳(秒)"`
}

// MonitorCpu CPU 指标。
type MonitorCpu struct {
	UsagePercent float64   `json:"usagePercent" dc:"总体占用率(%)"`
	PerCore      []float64 `json:"perCore"      dc:"每逻辑核占用率(%)"`
	CoresLogical int       `json:"coresLogical" dc:"逻辑核数"`
	CoresPhysical int      `json:"coresPhysical" dc:"物理核数"`
	ModelName    string    `json:"modelName"    dc:"CPU 型号"`
	Mhz          float64   `json:"mhz"          dc:"主频(MHz)"`
	Load1        float64   `json:"load1"        dc:"1 分钟负载"`
	Load5        float64   `json:"load5"        dc:"5 分钟负载"`
	Load15       float64   `json:"load15"       dc:"15 分钟负载"`
}

// MonitorMemory 内存指标 (物理内存)。
type MonitorMemory struct {
	Total        uint64  `json:"total"        dc:"总量(字节)"`
	Used         uint64  `json:"used"         dc:"已用(字节)"`
	Available    uint64  `json:"available"    dc:"可用(字节)"`
	UsedPercent  float64 `json:"usedPercent"  dc:"占用率(%)"`
}

// MonitorDisk 磁盘分区用量。
type MonitorDisk struct {
	Device      string  `json:"device"      dc:"设备名"`
	Mount       string  `json:"mount"       dc:"挂载点"`
	Fstype      string  `json:"fstype"      dc:"文件系统类型"`
	Total       uint64  `json:"total"       dc:"总量(字节)"`
	Used        uint64  `json:"used"        dc:"已用(字节)"`
	Free        uint64  `json:"free"        dc:"可用(字节)"`
	UsedPercent float64 `json:"usedPercent" dc:"占用率(%)"`
}

// MonitorGo Go 运行时指标。
type MonitorGo struct {
	GoVersion      string  `json:"goVersion"      dc:"编译用的 Go 版本"`
	Goroutines     int     `json:"goroutines"     dc:"当前 goroutine 数"`
	GoMaxProcs     int     `json:"goMaxProcs"     dc:"GOMAXPROCS"`
	HeapAlloc      uint64  `json:"heapAlloc"      dc:"堆已分配(字节)"`
	HeapSys        uint64  `json:"heapSys"        dc:"堆从 OS 申请(字节)"`
	StackSys       uint64  `json:"stackSys"       dc:"栈从 OS 申请(字节)"`
	Sys            uint64  `json:"sys"            dc:"运行时从 OS 申请总量(字节)"`
	NumGC          uint32  `json:"numGC"          dc:"GC 次数"`
	GCPauseTotalMs float64 `json:"gcPauseTotalMs" dc:"GC 累计停顿(毫秒)"`
	LastGCPauseMs  float64 `json:"lastGCPauseMs"  dc:"最近一次 GC 停顿(毫秒)"`
}

// MonitorMysqlReq MySQL 连接与运行状态。
type MonitorMysqlReq struct {
	g.Meta `path:"/system/monitor/mysql" tags:"SystemMonitor" method:"get" summary:"MySQL运行状态(仅超管)"`
}

// MonitorMysqlRes MySQL 状态响应。
type MonitorMysqlRes struct {
	Version            string `json:"version"            dc:"MySQL 版本"`
	UptimeSec          uint64 `json:"uptimeSec"          dc:"MySQL 已运行时长(秒)"`
	ThreadsConnected   int64  `json:"threadsConnected"   dc:"当前连接数"`
	ThreadsRunning     int64  `json:"threadsRunning"     dc:"活跃连接数(执行中)"`
	MaxUsedConnections int64  `json:"maxUsedConnections" dc:"历史最大连接数"`
	AbortedConnects    uint64 `json:"abortedConnects"    dc:"失败连接数(累计)"`
	SlowQueries        uint64 `json:"slowQueries"        dc:"慢查询数(累计)"`
	Queries            uint64 `json:"queries"            dc:"语句执行数(累计, Questions)"`
	BytesReceived      uint64 `json:"bytesReceived"      dc:"入流量(累计, 字节)"`
	BytesSent          uint64 `json:"bytesSent"          dc:"出流量(累计, 字节)"`
	// 连接池为应用侧视角 (go-sql-driver), 与 MySQL 服务端 Threads_* 互补
	Pool MonitorDbPool `json:"pool" dc:"应用侧连接池状态"`
}

// MonitorDbPool 应用侧数据库连接池状态。
type MonitorDbPool struct {
	Host          string `json:"host"          dc:"数据库节点"`
	Name          string `json:"name"          dc:"数据库名"`
	Open          int    `json:"open"          dc:"池中连接总数"`
	InUse         int    `json:"inUse"         dc:"使用中"`
	Idle          int    `json:"idle"          dc:"空闲"`
	WaitCount     int64  `json:"waitCount"     dc:"等待连接的请求数(累计)"`
	WaitDurationMs int64 `json:"waitDurationMs" dc:"累计等待时长(毫秒)"`
}

// MonitorRedisReq Redis 连接与运行状态。
type MonitorRedisReq struct {
	g.Meta `path:"/system/monitor/redis" tags:"SystemMonitor" method:"get" summary:"Redis运行状态(仅超管)"`
}

// MonitorRedisRes Redis 状态响应。
type MonitorRedisRes struct {
	Version             string  `json:"version"             dc:"Redis 版本"`
	UptimeSec           int64   `json:"uptimeSec"           dc:"Redis 已运行时长(秒)"`
	ConnectedClients    int64   `json:"connectedClients"    dc:"当前客户端连接数"`
	UsedMemory          uint64  `json:"usedMemory"          dc:"数据占用内存(字节)"`
	UsedMemoryHuman     string  `json:"usedMemoryHuman"     dc:"数据占用内存(可读)"`
	UsedMemoryPeakHuman string  `json:"usedMemoryPeakHuman" dc:"内存历史峰值(可读)"`
	TotalConnections    uint64  `json:"totalConnections"    dc:"累计连接数"`
	OpsPerSec           int64   `json:"opsPerSec"           dc:"每秒操作数"`
	KeyspaceHits        uint64  `json:"keyspaceHits"        dc:"键命中数(累计)"`
	KeyspaceMisses      uint64  `json:"keyspaceMisses"      dc:"键未命中数(累计)"`
	HitRatePercent      float64 `json:"hitRatePercent"      dc:"命中率(%), 无访问时为 -1"`
	DbSize              int64   `json:"dbSize"              dc:"当前库键数量"`
}

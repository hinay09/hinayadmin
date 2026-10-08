/**
 * 服务器监控 API (仅超管)。
 */
import { useRequest } from '~/composables/useRequest'

/** 服务器指标 */
export interface ServerMonitor {
  host: {
    hostname: string
    os: string
    platform: string
    kernelArch: string
    kernelVersion: string
    bootTime: number
    uptimeSec: number
    appUptimeSec: number
    serverTime: number
  }
  cpu: {
    usagePercent: number
    perCore: number[]
    coresLogical: number
    coresPhysical: number
    modelName: string
    mhz: number
    load1: number
    load5: number
    load15: number
  }
  memory: {
    total: number
    used: number
    available: number
    usedPercent: number
  }
  disks: Array<{
    device: string
    mount: string
    fstype: string
    total: number
    used: number
    free: number
    usedPercent: number
  }>
  go: {
    goVersion: string
    goroutines: number
    goMaxProcs: number
    heapAlloc: number
    heapSys: number
    stackSys: number
    sys: number
    numGC: number
    gcPauseTotalMs: number
    lastGCPauseMs: number
  }
}

/** MySQL 运行状态 */
export interface MysqlMonitor {
  version: string
  uptimeSec: number
  threadsConnected: number
  threadsRunning: number
  maxUsedConnections: number
  abortedConnects: number
  slowQueries: number
  queries: number
  bytesReceived: number
  bytesSent: number
  pool: {
    host: string
    name: string
    open: number
    inUse: number
    idle: number
    waitCount: number
    waitDurationMs: number
  }
}

/** Redis 运行状态 */
export interface RedisMonitor {
  version: string
  uptimeSec: number
  connectedClients: number
  usedMemory: number
  usedMemoryHuman: string
  usedMemoryPeakHuman: string
  totalConnections: number
  opsPerSec: number
  keyspaceHits: number
  keyspaceMisses: number
  hitRatePercent: number
  dbSize: number
}

export function useMonitorApi() {
  const r = useRequest()
  const base = '/system/monitor'
  return {
    server: () => r.get<ServerMonitor>(`${base}/server`, undefined, { silent: true }),
    mysql: () => r.get<MysqlMonitor>(`${base}/mysql`, undefined, { silent: true }),
    redis: () => r.get<RedisMonitor>(`${base}/redis`, undefined, { silent: true }),
  }
}

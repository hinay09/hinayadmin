<script setup lang="ts">
/**
 * Cron 可视化生成器 (gcron 6 位表达式: 秒 分 时 日 月 周, 无年域)。
 * 每域四种模式: 每(单位) / 周期(X-Y) / 间隔(从X起每N) / 指定(多选)。
 * 打开时回显解析现有表达式; 不支持的高级语法会提示并回退默认。
 * 参考 toolsjy.com/cron 的交互形态。
 */
import { ref, reactive, computed, watch } from 'vue'

const props = defineProps<{ modelValue: boolean; cron: string }>()
const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'apply', cron: string): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v),
})

// ---- 字段定义 (顺序 = 表达式位置) ----
interface FieldDef {
  key: string
  label: string
  min: number
  max: number
  valueLabel?: (n: number) => string
}
const WEEK_CN = ['日', '一', '二', '三', '四', '五', '六']
const FIELDS: FieldDef[] = [
  { key: 'sec', label: '秒', min: 0, max: 59 },
  { key: 'min', label: '分钟', min: 0, max: 59 },
  { key: 'hour', label: '小时', min: 0, max: 23 },
  { key: 'day', label: '日', min: 1, max: 31 },
  { key: 'month', label: '月', min: 1, max: 12 },
  { key: 'week', label: '周', min: 0, max: 6, valueLabel: n => `周${WEEK_CN[n]}` },
]

type Mode = 'every' | 'range' | 'interval' | 'appoint'
interface FieldState {
  mode: Mode
  rangeFrom: number
  rangeTo: number
  intervalFrom: number
  intervalEvery: number
  selected: number[]
}
const defaults = (f: FieldDef): FieldState => ({
  mode: 'every',
  rangeFrom: f.min,
  rangeTo: f.max,
  intervalFrom: f.min,
  intervalEvery: f.min === 1 ? 1 : f.min === 0 ? 1 : 1,
  selected: [f.min],
})

const state = reactive<Record<string, FieldState>>(
  Object.fromEntries(FIELDS.map(f => [f.key, defaults(f)])),
)
const unsupported = ref(false)

/* ---- 常用频率一键模板 ----
 * "每分钟/每小时/每天..." 类定时的高位字段须固定为具体值 (如 每分钟 = 0 * * * * *,
 * 秒固定 0), 手动逐域配置容易漏。此处一键铺底, 铺完仍可在下方继续微调。 */
const setEvery = (key: string) => { state[key].mode = 'every' }
const setAppoint = (key: string, vals: number[]) => {
  state[key].mode = 'appoint'
  state[key].selected = vals
}

function applyFrequency(kind: 'perSec' | 'perMin' | 'perHour' | 'daily' | 'weekly' | 'monthly') {
  FIELDS.forEach(f => setEvery(f.key))
  switch (kind) {
    case 'perSec':
      return // * * * * * *
    case 'perMin':
      setAppoint('sec', [0])
      return // 0 * * * * *
    case 'perHour':
      setAppoint('sec', [0]); setAppoint('min', [0])
      return // 0 0 * * * *
    case 'daily':
      setAppoint('sec', [0]); setAppoint('min', [0]); setAppoint('hour', [0])
      return // 0 0 0 * * *  (默认 0 点, 可在下方改小时)
    case 'weekly':
      setAppoint('sec', [0]); setAppoint('min', [0]); setAppoint('hour', [0])
      setAppoint('week', [1])
      return // 0 0 0 * * 1  (默认周一 0 点, 可改周几/小时)
    case 'monthly':
      setAppoint('sec', [0]); setAppoint('min', [0]); setAppoint('hour', [0])
      setAppoint('day', [1])
      return // 0 0 0 1 * *  (默认每月 1 号 0 点, 可改日期/小时)
  }
}

// ---- 打开时回显解析 ----
watch(visible, (v) => {
  if (!v) return
  unsupported.value = false
  const parts = (props.cron || '').trim().split(/\s+/)
  if (parts.length !== 6 || parts.some(p => !p)) {
    if (props.cron) unsupported.value = true
    return // 空/非法: 保持面板默认值
  }
  FIELDS.forEach((f, i) => parseField(parts[i], f))
})

function parseField(expr: string, f: FieldDef) {
  const s = state[f.key]
  if (expr === '*') {
    s.mode = 'every'
    return
  }
  let m = expr.match(/^(\d+)-(\d+)$/)
  if (m) {
    s.mode = 'range'
    s.rangeFrom = clamp(+m[1], f)
    s.rangeTo = clamp(+m[2], f)
    return
  }
  m = expr.match(/^(\d+)\/(\d+)$/)
  if (m) {
    s.mode = 'interval'
    s.intervalFrom = clamp(+m[1], f)
    s.intervalEvery = Math.max(1, +m[2])
    return
  }
  m = expr.match(/^\*\/(\d+)$/)
  if (m) {
    s.mode = 'interval'
    s.intervalFrom = f.min
    s.intervalEvery = Math.max(1, +m[1])
    return
  }
  m = expr.match(/^(\d+(,\d+)+|\d+)$/)
  if (m) {
    s.mode = 'appoint'
    s.selected = expr.split(',').map(Number).filter(n => n >= f.min && n <= f.max)
    return
  }
  unsupported.value = true // L/W/# 等高级语法
}

function clamp(n: number, f: FieldDef) {
  return Math.min(f.max, Math.max(f.min, n))
}

// ---- 组装 ----
function composeField(f: FieldDef): string {
  const s = state[f.key]
  switch (s.mode) {
    case 'every': return '*'
    case 'range': return `${s.rangeFrom}-${s.rangeTo}`
    case 'interval': return `${s.intervalFrom}/${s.intervalEvery}`
    case 'appoint': return s.selected.length ? [...s.selected].sort((a, b) => a - b).join(',') : '*'
  }
}
const composed = computed(() => FIELDS.map(composeField).join(' '))

// ---- 中文描述预览 ----
function descField(f: FieldDef): string {
  const s = state[f.key]
  const name = (n: number) => f.valueLabel ? f.valueLabel(n) : String(n)
  switch (s.mode) {
    case 'every': return ''
    case 'range': return `${name(s.rangeFrom)}到${name(s.rangeTo)}的每${f.label}`
    case 'interval': return `从${name(s.intervalFrom)}${f.label}起每 ${s.intervalEvery} ${f.label}`
    case 'appoint': return `指定的${f.label}(${s.selected.slice().sort((a, b) => a - b).map(name).join('/')})`
  }
}
const description = computed(() => {
  const parts = FIELDS.map(f => ({ f, d: descField(f) })).filter(x => x.d)
  if (!parts.length) return '每秒执行'
  // 组合阅读顺序: 月 -> 周/日 -> 时 -> 分 -> 秒
  const order = ['month', 'week', 'day', 'hour', 'min', 'sec']
  return parts.sort((a, b) => order.indexOf(a.f.key) - order.indexOf(b.f.key))
    .map(x => x.d).join(', ')
})

function opts(f: FieldDef) {
  return Array.from({ length: f.max - f.min + 1 }, (_, i) => f.min + i)
}

function handleApply() {
  emit('apply', composed.value)
  visible.value = false
}
</script>

<template>
  <el-dialog v-model="visible" title="Cron 表达式生成" width="720px" top="5vh">
    <el-alert v-if="unsupported" type="warning" :closable="false" style="margin-bottom:10px"
      title="原表达式含生成器不支持的高级语法 (如 L/W/#), 面板已回退默认值; 直接保存将覆盖原表达式" />

    <!-- 字段示意 -->
    <el-collapse style="margin-bottom:12px">
      <el-collapse-item title="表达式字段说明 (点击展开)">
        <table class="cron-legend">
          <tr><th>位置</th><th>字段</th><th>允许值</th><th>说明</th></tr>
          <tr><td>1</td><td>秒</td><td>0-59</td><td rowspan="6" class="legend-sym">
            <code>*</code> 任意值 &nbsp; <code>,</code> 枚举多个值<br>
            <code>-</code> 范围 (从-到) &nbsp; <code>/</code> 步进 (起始/间隔)<br>
            <span class="legend-note">本系统为 GoFrame gcron 的 6 位表达式 (含秒, 无年域)</span>
          </td></tr>
          <tr><td>2</td><td>分钟</td><td>0-59</td></tr>
          <tr><td>3</td><td>小时</td><td>0-23</td></tr>
          <tr><td>4</td><td>日</td><td>1-31</td></tr>
          <tr><td>5</td><td>月</td><td>1-12</td></tr>
          <tr><td>6</td><td>周</td><td>0-6 (0=周日)</td></tr>
        </table>
      </el-collapse-item>
    </el-collapse>

    <!-- 常用频率 -->
    <div class="freq-row">
      <span class="freq-label">常用频率</span>
      <el-button size="small" @click="applyFrequency('perSec')">每秒</el-button>
      <el-button size="small" @click="applyFrequency('perMin')">每分钟</el-button>
      <el-button size="small" @click="applyFrequency('perHour')">每小时</el-button>
      <el-button size="small" @click="applyFrequency('daily')">每天</el-button>
      <el-button size="small" @click="applyFrequency('weekly')">每周</el-button>
      <el-button size="small" @click="applyFrequency('monthly')">每月</el-button>
      <span class="freq-tip">一键铺底后可在下方继续微调 (如每周改周几/几点)</span>
    </div>

    <!-- 六域配置 -->
    <div class="field-grid">
      <div v-for="f in FIELDS" :key="f.key" class="field-card">
        <div class="field-title">{{ f.label }}</div>
        <el-radio-group v-model="state[f.key].mode" size="small">
          <el-radio value="every">每{{ f.label }}</el-radio>
          <el-radio value="range">周期</el-radio>
          <el-radio value="interval">间隔</el-radio>
          <el-radio value="appoint">指定</el-radio>
        </el-radio-group>

        <div v-if="state[f.key].mode === 'range'" class="mode-row">
          从
          <el-select v-model="state[f.key].rangeFrom" size="small" style="width:86px">
            <el-option v-for="n in opts(f)" :key="n" :value="n" :label="f.valueLabel ? f.valueLabel(n) : n" />
          </el-select>
          到
          <el-select v-model="state[f.key].rangeTo" size="small" style="width:86px">
            <el-option v-for="n in opts(f)" :key="n" :value="n" :label="f.valueLabel ? f.valueLabel(n) : n" />
          </el-select>
        </div>

        <div v-else-if="state[f.key].mode === 'interval'" class="mode-row">
          从
          <el-select v-model="state[f.key].intervalFrom" size="small" style="width:86px">
            <el-option v-for="n in opts(f)" :key="n" :value="n" :label="f.valueLabel ? f.valueLabel(n) : n" />
          </el-select>
          开始, 每
          <el-select v-model="state[f.key].intervalEvery" size="small" style="width:76px">
            <el-option v-for="n in opts(f)" :key="n" :value="n" :label="n" />
          </el-select>
          {{ f.label }}
        </div>

        <div v-else-if="state[f.key].mode === 'appoint'" class="mode-row appoint">
          <el-checkbox-group v-model="state[f.key].selected" size="small">
            <el-checkbox v-for="n in opts(f)" :key="n" :value="n">
              {{ f.valueLabel ? f.valueLabel(n) : n }}
            </el-checkbox>
          </el-checkbox-group>
        </div>
      </div>
    </div>

    <!-- 预览 -->
    <div class="preview">
      <div class="preview-expr">
        <span class="preview-label">表达式</span>
        <code>{{ composed }}</code>
      </div>
      <div class="preview-desc">
        <span class="preview-label">解读</span>{{ description }}
      </div>
    </div>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="handleApply">生成并填入</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.cron-legend {
  border-collapse: collapse;
  font-size: 12px;
  width: 100%;
}
.cron-legend th, .cron-legend td {
  border: 1px solid var(--el-border-color-lighter);
  padding: 4px 10px;
  text-align: left;
}
.legend-sym {
  vertical-align: middle;
  line-height: 1.8;
}
.legend-note {
  color: var(--el-text-color-secondary);
}

.freq-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 12px;
}
.freq-label {
  font-weight: 600;
  font-size: 13px;
  margin-right: 6px;
}
.freq-row .el-button + .el-button {
  margin-left: 0;
}
.freq-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-left: 8px;
}

.field-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.field-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 10px 12px;
}
.field-title {
  font-weight: 600;
  margin-bottom: 6px;
  color: var(--el-text-color-primary);
}
.mode-row {
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  font-size: 13px;
}
.mode-row.appoint {
  max-height: 110px;
  overflow-y: auto;
}

.preview {
  margin-top: 12px;
  background: var(--el-fill-color-light);
  border-radius: 6px;
  padding: 10px 12px;
}
.preview-expr {
  display: flex;
  align-items: center;
  gap: 10px;
}
.preview-expr code {
  font-family: 'SF Mono', Menlo, Consolas, monospace;
  font-size: 15px;
  color: var(--el-color-primary);
  font-weight: 600;
}
.preview-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
}
.preview-desc {
  margin-top: 6px;
  font-size: 13px;
  color: var(--el-text-color-regular);
}
</style>

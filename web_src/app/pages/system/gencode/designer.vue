<script setup lang="ts">
/**
 * 表单设计器 (代码生成): 基于共享拖拽壳 (FormDesignerShell) 产出字段 schema,
 * 一键套用到「代码生成」页的列配置 (控件/必填覆盖, 复用 crudgen 产出 Vue 页面)。
 *
 * schema 契约: [{ prop, label, type, required, dictCode?, relMod? }]
 * 持久化: localStorage 草稿; 「应用到代码生成」经 sessionStorage
 * (hinay_form_schema) 传递, gencode 页选表后按列名匹配套用。
 */
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, MagicStick, DocumentCopy, Upload, Position } from '@element-plus/icons-vue'
import { useGencodeApi } from '~/composables/useApi'
import FormDesignerShell, { type DesignerField, type DesignerPaletteItem } from '~/components/FormDesignerShell.vue'

definePageMeta({ title: '表单设计器' })
const router = useRouter()
defineOptions({ name: 'system-gencode-designer' })

const api = useGencodeApi()

/** 控件清单 (kind 与后端 crudgen.ValidKinds 对齐) */
const palette: DesignerPaletteItem[] = [
  { type: 'input', label: '单行输入', hint: '常规文本' },
  { type: 'textarea', label: '多行文本', hint: '备注/描述' },
  { type: 'number', label: '数字输入', hint: '数量/金额' },
  { type: 'switch', label: '开关', hint: '状态 0/1' },
  { type: 'date', label: '日期时间', hint: '时间选择' },
  { type: 'dict', label: '字典下拉', hint: '选项来自字典管理' },
  { type: 'relation', label: '关联下拉', hint: '选项来自其他模块 /all' },
]

const STORAGE_KEY = 'hinay_form_designer_draft'
const SCHEMA_KEY = 'hinay_form_schema'

const fields = ref<DesignerField[]>([])
const ioVisible = ref(false)
const ioText = ref('')
const ioMode = ref<'import' | 'export'>('export')
const tableLoading = ref(false)
const tables = ref<Array<{ tableName: string, tableComment: string }>>([])
const importTableVisible = ref(false)
const importTableName = ref('')
const importColLoading = ref(false)

/* ---- schema 导入/导出 ---- */
function openExport() {
  ioMode.value = 'export'
  ioText.value = JSON.stringify(fields.value, null, 2)
  ioVisible.value = true
}

function openImport() {
  ioMode.value = 'import'
  ioText.value = ''
  ioVisible.value = true
}

function confirmImport() {
  try {
    const parsed = JSON.parse(ioText.value)
    if (!Array.isArray(parsed)) throw new Error('顶层应为数组')
    const cleaned: DesignerField[] = []
    const nameSet = new Set<string>()
    for (const f of parsed) {
      const prop = String(f.prop || '').trim()
      if (!/^[a-z][a-z0-9_]*$/.test(prop)) throw new Error(`字段名不合法: ${f.prop}`)
      if (nameSet.has(prop)) throw new Error(`字段名重复: ${prop}`)
      nameSet.add(prop)
      const type = palette.find(p => p.type === f.type)?.type
      if (!type) throw new Error(`未知控件类型: ${f.type}`)
      cleaned.push({
        prop,
        label: String(f.label || prop),
        type,
        required: !!f.required,
        ...(type === 'dict' ? { dictCode: String(f.dictCode || '') } : {}),
        ...(type === 'relation' ? { relMod: String(f.relMod || '') } : {}),
      })
    }
    fields.value = cleaned
    ioVisible.value = false
    ElMessage.success(`已导入 ${cleaned.length} 个字段`)
  }
  catch (e: any) {
    ElMessage.error(`schema 不合法: ${e.message}`)
  }
}

async function copySchema() {
  try {
    await navigator.clipboard.writeText(ioText.value)
    ElMessage.success('已复制到剪贴板')
  }
  catch {
    ElMessage.warning('复制失败, 请手动选择文本复制')
  }
}

/* ---- 从数据表导入 (复用 gencode 列接口 + 类型推导) ---- */
async function openImportTable() {
  importTableVisible.value = true
  if (tables.value.length) return
  tableLoading.value = true
  try {
    const res = await api.tables()
    tables.value = res.list || []
  }
  finally {
    tableLoading.value = false
  }
}

/** 与后端 detectKind 对齐的客户端推导 (导入预填, 可在属性面板改) */
function detectKind(name: string, dataType: string, comment: string): { type: string, dictCode?: string, relMod?: string } {
  if (/字典[:：]\s*([A-Za-z0-9_-]+)/.test(comment)) {
    return { type: 'dict', dictCode: comment.match(/字典[:：]\s*([A-Za-z0-9_-]+)/)![1] }
  }
  if (/关联[:：]\s*([a-z][a-z0-9_]*)/.test(comment)) {
    return { type: 'relation', relMod: comment.match(/关联[:：]\s*([a-z][a-z0-9_]*)/)![1] }
  }
  if (name === 'status' || name.endsWith('_status')) return { type: 'switch' }
  if (/remark|content|description|desc/.test(name)) return { type: 'textarea' }
  if (name.endsWith('_id')) return { type: 'relation', relMod: name.replace(/_id$/, '') }
  const t = dataType.toUpperCase()
  if (['BIGINT', 'INT', 'TINYINT', 'SMALLINT', 'MEDIUMINT', 'DECIMAL', 'FLOAT', 'DOUBLE'].includes(t)) return { type: 'number' }
  if (['DATETIME', 'TIMESTAMP', 'DATE'].includes(t)) return { type: 'date' }
  return { type: 'input' }
}

async function confirmImportTable() {
  if (!importTableName.value) return
  importColLoading.value = true
  try {
    const res = await api.columns(importTableName.value)
    const cols: any[] = (res.list || []).filter((c: any) => !c.skip)
    fields.value = cols.map(c => ({
      prop: c.name,
      label: c.comment || c.name,
      required: false,
      ...detectKind(c.name, c.dataType, c.comment || ''),
    } as DesignerField))
    importTableVisible.value = false
    ElMessage.success(`已从 ${importTableName.value} 导入 ${fields.value.length} 个字段`)
  }
  finally {
    importColLoading.value = false
  }
}

/* ---- 应用到代码生成 ---- */
function applyToGencode() {
  if (!fields.value.length) {
    ElMessage.warning('画布为空')
    return
  }
  // 字段名与数据源合法性最终把关
  for (const f of fields.value) {
    if (!/^[a-z][a-z0-9_]*$/.test(f.prop)) {
      ElMessage.error(`字段名不合法: ${f.prop} (小写字母开头的 snake_case)`)
      return
    }
    if (f.type === 'dict' && !f.dictCode) {
      ElMessage.error(`字典字段 ${f.prop} 未填字典编码`)
      return
    }
    if (f.type === 'relation' && !f.relMod) {
      ElMessage.error(`关联字段 ${f.prop} 未填关联模块`)
      return
    }
  }
  sessionStorage.setItem(SCHEMA_KEY, JSON.stringify(fields.value))
  ElMessage.success('schema 已暂存, 请在代码生成页选择数据表 (列名匹配自动套用)')
  router.push('/system/gencode')
}

/* ---- 草稿持久化 ---- */
function saveDraft() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(fields.value))
}

function clearAll() {
  ElMessageBox.confirm('清空画布所有字段?', '提示', { type: 'warning' }).then(() => {
    fields.value = []
  }).catch(() => {})
}

onMounted(() => {
  const raw = localStorage.getItem(STORAGE_KEY)
  if (raw) {
    try {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed) && parsed.length) {
        fields.value = parsed
        ElMessage.info(`已恢复草稿 (${parsed.length} 个字段)`)
      }
    }
    catch {}
  }
})
</script>

<template>
  <div class="designer-page">
    <el-card :body-style="{ padding: '10px 14px' }">
      <div class="toolbar-row">
        <span class="toolbar-title">表单设计器</span>
        <span class="toolbar-sub">拖拽组件到画布 → 配置属性 → 应用到代码生成 (字段名与表列名匹配)</span>
        <div class="toolbar-actions">
          <el-button :icon="MagicStick" @click="openImportTable">从数据表导入</el-button>
          <el-button :icon="DocumentCopy" @click="openExport">导出 schema</el-button>
          <el-button :icon="Upload" @click="openImport">导入 schema</el-button>
          <el-button type="danger" plain :icon="Delete" :disabled="!fields.length" @click="clearAll">清空</el-button>
          <el-button type="primary" :icon="Position" :disabled="!fields.length" @click="applyToGencode">应用到代码生成</el-button>
        </div>
      </div>
    </el-card>

    <el-card>
      <FormDesignerShell
        v-model="fields"
        :palette="palette"
        prop-placeholder="snake_case, 须与表列名一致"
      >
        <template #extra="{ field, patch }">
          <el-form-item v-if="field.type === 'dict'" label="字典编码">
            <el-input :model-value="field.dictCode" placeholder="字典管理的类型编码" @update:model-value="(v: string) => patch({ dictCode: v })" />
          </el-form-item>
          <el-form-item v-if="field.type === 'relation'" label="关联模块">
            <el-input :model-value="field.relMod" placeholder="如 post → usePostApi().all()" @update:model-value="(v: string) => patch({ relMod: v })" />
          </el-form-item>
        </template>
      </FormDesignerShell>
      <div class="draft-row">
        <el-button plain @click="saveDraft">保存草稿 (localStorage)</el-button>
      </div>
    </el-card>

    <!-- schema 导入/导出 -->
    <el-dialog v-model="ioVisible" :title="ioMode === 'export' ? '导出 schema' : '导入 schema'" width="640px">
      <div v-if="ioMode === 'export'" style="margin-bottom:8px">
        <el-button size="small" :icon="DocumentCopy" @click="copySchema">复制</el-button>
      </div>
      <el-input
        v-model="ioText"
        type="textarea"
        :rows="14"
        :placeholder="ioMode === 'import' ? '粘贴 schema JSON 数组' : ''"
        spellcheck="false"
      />
      <template #footer>
        <el-button @click="ioVisible = false">关闭</el-button>
        <el-button v-if="ioMode === 'import'" type="primary" @click="confirmImport">导入</el-button>
      </template>
    </el-dialog>

    <!-- 从数据表导入 -->
    <el-dialog v-model="importTableVisible" title="从数据表导入字段" width="480px">
      <el-select
        v-model="importTableName"
        v-loading="tableLoading"
        filterable
        placeholder="选择数据表"
        style="width:100%"
      >
        <el-option
          v-for="t in tables"
          :key="t.tableName"
          :value="t.tableName"
          :label="t.tableComment ? `${t.tableName} (${t.tableComment})` : t.tableName"
        />
      </el-select>
      <div class="import-table-tip">
        按列注释指令 (字典:code / 关联:mod) 与列类型自动推导控件; 导入后可在属性面板逐字段调整
      </div>
      <template #footer>
        <el-button @click="importTableVisible = false">取消</el-button>
        <el-button type="primary" :loading="importColLoading" :disabled="!importTableName" @click="confirmImportTable">导入</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.designer-page {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.toolbar-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.toolbar-title {
  font-weight: 600;
}

.toolbar-sub {
  flex: 1;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.draft-row {
  margin-top: 10px;
}

.import-table-tip {
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}
</style>

<script setup lang="ts">
/**
 * CMS 文章编辑 (整页): wangeditor 正文 + 封面/栏目/摘要/标签。
 * - query.id 存在 = 编辑 (加载详情回显), 否则新增;
 * - 封面存持久引用 (ref), 预览用展示 URL —— 切换存储/预签名过期不影响已存文章;
 * - 保存并发布: 新增直接 status=1; 编辑先保存再调发布动作。
 */
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { ArrowLeft, Picture, Delete } from '@element-plus/icons-vue'
import {
  useCmsArticleApi,
  useCmsCategoryApi,
  type CmsCategoryItem,
} from '~/composables/useApi'

definePageMeta({ title: '文章编辑' })
defineOptions({ name: 'cms-article-edit' })

const api = useCmsArticleApi()
const categoryApi = useCmsCategoryApi()
const route = useRoute()
const router = useRouter()

const articleId = computed(() => Number(route.query.id) || 0)
const isEdit = computed(() => articleId.value > 0)
/** 当前文章状态 (详情加载后更新; 新增默认 0=草稿) —— 已发布时不再提供「保存并发布」 */
const currentStatus = ref<number>(0)
const isPublished = computed(() => isEdit.value && currentStatus.value === 1)

const formRef = ref<FormInstance>()
const form = reactive({
  categoryId: 0,
  title: '',
  cover: '', // 持久引用 (上传响应 ref)
  coverView: '', // 展示 URL (预览用)
  summary: '',
  content: '',
  tags: '',
  top: 0,
})
const rules = {
  title: [{ required: true, message: '请输入文章标题', trigger: 'blur' }],
}

const categories = ref<CmsCategoryItem[]>([])
const loading = ref(false)
const saving = ref(false)
const publishing = ref(false)

async function loadCategories() {
  try {
    const res = await categoryApi.all()
    categories.value = res.list || []
  }
  catch {}
}

async function loadDetail() {
  loading.value = true
  try {
    const res = await api.detail(articleId.value)
    const d = res.detail
    if (!d) return
    Object.assign(form, {
      categoryId: d.categoryId,
      title: d.title,
      cover: d.coverRef || '',
      coverView: d.cover || '',
      summary: d.summary || '',
      content: d.content || '',
      tags: d.tags || '',
      top: d.top || 0,
    })
    currentStatus.value = d.status ?? 0
  }
  finally { loading.value = false }
}

function handleCoverUploaded(res: any) {
  // 存持久引用, 展示用 URL 仅作预览
  form.cover = res.ref || res.url
  form.coverView = res.url
}

function removeCover() {
  form.cover = ''
  form.coverView = ''
}

async function save(publish: boolean) {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  const busy = publish ? publishing : saving
  busy.value = true
  try {
    if (isEdit.value) {
      await api.update(articleId.value, {
        categoryId: form.categoryId,
        title: form.title,
        cover: form.cover,
        summary: form.summary,
        content: form.content,
        tags: form.tags,
        top: form.top,
      })
      if (publish) {
        await api.publish(articleId.value) // 后端幂等: 已发布重入同样成功
        currentStatus.value = 1
      }
      ElMessage.success(publish ? '已保存并发布' : '保存成功')
    }
    else {
      const res = await api.create({
        categoryId: form.categoryId,
        title: form.title,
        cover: form.cover,
        summary: form.summary,
        content: form.content,
        tags: form.tags,
        top: form.top,
        status: publish ? 1 : 0,
      })
      currentStatus.value = publish ? 1 : 0
      ElMessage.success(publish ? '已发布' : '已保存草稿')
      // 原地转为编辑态, 连续保存不会重复建稿
      router.replace(`/cms/article/edit?id=${res.id}`)
    }
  }
  catch {} finally { busy.value = false }
}

function back() {
  router.push('/cms/article')
}

onMounted(() => {
  loadCategories()
  if (isEdit.value) loadDetail()
})
</script>

<template>
  <div class="page" v-loading="loading">
    <el-card>
      <template #header>
        <div class="header">
          <el-button :icon="ArrowLeft" link @click="back">返回列表</el-button>
          <span class="header-title">{{ isEdit ? '编辑文章' : '写文章' }}</span>
        </div>
      </template>

      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item label="标题" prop="title">
          <el-input v-model="form.title" maxlength="200" show-word-limit placeholder="请输入文章标题" />
        </el-form-item>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="栏目">
              <el-select v-model="form.categoryId" placeholder="未分类" clearable style="width:100%">
                <el-option label="未分类" :value="0" />
                <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="标签">
              <el-input v-model="form.tags" maxlength="200" placeholder="多个标签用英文逗号分隔" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="封面">
          <div class="cover-row">
            <div v-if="form.coverView" class="cover-preview">
              <el-image :src="form.coverView" fit="cover"
                :preview-src-list="[form.coverView]" preview-teleported
                style="width:160px;height:90px;border-radius:6px" />
              <el-button :icon="Delete" circle size="small" type="danger" class="cover-del" @click="removeCover" />
            </div>
            <FileUploader accept="image/*" @success="handleCoverUploaded">
              <template #default="{ uploading }">
                <el-button :icon="Picture" :loading="uploading">{{ form.coverView ? '更换封面' : '上传封面' }}</el-button>
              </template>
            </FileUploader>
            <span class="cover-tip">建议 16:9 比例; 点击封面可预览大图</span>
          </div>
        </el-form-item>

        <el-form-item label="摘要">
          <el-input v-model="form.summary" type="textarea" :rows="2" maxlength="500" show-word-limit
            placeholder="列表展示用摘要, 可留空" />
        </el-form-item>

        <el-form-item label="正文">
          <ClientOnly>
            <RichEditor v-model="form.content" :height="480" placeholder="请输入正文, 支持图片/视频上传…" />
            <template #fallback>
              <el-skeleton :rows="6" animated style="height:480px" />
            </template>
          </ClientOnly>
        </el-form-item>

        <el-form-item label="置顶">
          <el-switch v-model="form.top" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>

      <div class="footer">
        <el-button @click="back">取消</el-button>
        <!-- 已发布的文章没有「再发布」语义: 只留保存 (状态不变); 新稿/草稿/下架稿才提供发布 -->
        <el-button v-if="!isPublished" type="primary" plain :loading="publishing" @click="save(true)">保存并发布</el-button>
        <el-button type="primary" :loading="saving" @click="save(false)">
          {{ isEdit ? '保存' : '保存草稿' }}
        </el-button>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-title {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.cover-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.cover-preview {
  position: relative;
  display: inline-block;
}

.cover-del {
  position: absolute;
  top: -8px;
  right: -8px;
}

.cover-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 8px;
}
</style>

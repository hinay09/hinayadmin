// Package crudgen 模板定义。 (CLI 与网页端代码生成共用)
package crudgen

// ---------------------------------------------------------------------------
// 模板内容
// ---------------------------------------------------------------------------

const apiTmpl = `// Package v1 {{.Title}}接口。 (crudgen 生成)
package v1

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"

	"hinay.cn/admin/utility/response"
)

// I{{.Entity}}V1 控制器接口 (gf gen ctrl 约定)。
type I{{.Entity}}V1 interface {
	{{.Entity}}List(ctx context.Context, req *{{.Entity}}ListReq) (res *{{.Entity}}ListRes, err error)
	{{.Entity}}Create(ctx context.Context, req *{{.Entity}}CreateReq) (res *{{.Entity}}CreateRes, err error)
	{{.Entity}}Update(ctx context.Context, req *{{.Entity}}UpdateReq) (res *{{.Entity}}UpdateRes, err error)
	{{.Entity}}Delete(ctx context.Context, req *{{.Entity}}DeleteReq) (res *{{.Entity}}DeleteRes, err error)
}

// {{.Entity}}ListReq 分页列表。
type {{.Entity}}ListReq struct {
	g.Meta   {{bt}}path:"/{{.Mod}}" tags:"{{.Tag}}" method:"get" summary:"{{.Title}}分页列表"{{bt}}
	Keyword  string {{bt}}json:"keyword" in:"query" dc:"关键词模糊"{{bt}}
	Page     int    {{bt}}json:"page"     in:"query" d:"1"  dc:"页码"{{bt}}
	PageSize int    {{bt}}json:"pageSize" in:"query" d:"10" dc:"页大小"{{bt}}
}

// {{.Entity}}Item 列表项。
type {{.Entity}}Item struct {
	Id        uint64      {{bt}}json:"id"{{bt}}
{{- range .Columns}}
	{{.GoName}} {{.GoType}} {{bt}}json:"{{camel .GoName}}"{{bt}}
{{- end}}
	CreatedAt *gtime.Time {{bt}}json:"createdAt"{{bt}}
}

// {{.Entity}}ListRes 列表响应。
type {{.Entity}}ListRes response.PageResult

// {{.Entity}}CreateReq 新增。
type {{.Entity}}CreateReq struct {
	g.Meta {{bt}}path:"/{{.Mod}}" tags:"{{.Tag}}" method:"post" summary:"新增{{.Title}}"{{bt}}
{{- range .FormColumns}}
	{{.GoName}} {{.GoType}} {{bt}}json:"{{camel .GoName}}"{{bt}}
{{- end}}
}

// {{.Entity}}CreateRes 新增响应。
type {{.Entity}}CreateRes struct{}

// {{.Entity}}UpdateReq 修改。
type {{.Entity}}UpdateReq struct {
	g.Meta {{bt}}path:"/{{.Mod}}/:id" tags:"{{.Tag}}" method:"put" summary:"修改{{.Title}}"{{bt}}
	Id     uint64 {{bt}}json:"id" in:"path" v:"min:1"{{bt}}
{{- range .FormColumns}}
	{{.GoName}} {{.GoType}} {{bt}}json:"{{camel .GoName}}"{{bt}}
{{- end}}
}

// {{.Entity}}UpdateRes 修改响应。
type {{.Entity}}UpdateRes struct{}

// {{.Entity}}DeleteReq 删除 (软删)。
type {{.Entity}}DeleteReq struct {
	g.Meta {{bt}}path:"/{{.Mod}}/:id" tags:"{{.Tag}}" method:"delete" summary:"删除{{.Title}}"{{bt}}
	Id     uint64 {{bt}}json:"id" in:"path" v:"min:1"{{bt}}
}

// {{.Entity}}DeleteRes 删除响应。
type {{.Entity}}DeleteRes struct{}
`

const apiAliasTmpl = `// Package {{.Mod}} {{.Title}}控制器接口别名。 (crudgen 生成)
package {{.Mod}}

import v1 "hinay.cn/admin/api/{{.Mod}}/v1"

// I{{.Entity}}V1 与 v1 接口等价 (gf gen ctrl 的包级别名约定)。
type I{{.Entity}}V1 = v1.I{{.Entity}}V1
`

const ctrlNewTmpl = `// {{.Entity}} 控制器注册。 (crudgen 生成)
package {{.Mod}}

import (
	"hinay.cn/admin/api/{{.Mod}}"
)

type ControllerV1 struct{}

func NewV1() {{.Mod}}.I{{.Entity}}V1 {
	return &ControllerV1{}
}
`

const ctrlMethodTmpl = `package {{.Mod}}

import (
	"context"

	v1 "hinay.cn/admin/api/{{.Mod}}/v1"
	"hinay.cn/admin/internal/service"
)

func (c *ControllerV1) {{.Entity}}{{.Act}}(ctx context.Context, req *v1.{{.Entity}}{{.Act}}Req) (res *v1.{{.Entity}}{{.Act}}Res, err error) {
	return service.{{.Entity}}().{{.Act}}(ctx, req)
}
`

const svcTmpl = `// Package service {{.Title}}服务接口。 (crudgen 生成)
package service

import (
	"context"

	v1 "hinay.cn/admin/api/{{.Mod}}/v1"
)

type I{{.Entity}} interface {
	// List 分页列表。
	List(ctx context.Context, req *v1.{{.Entity}}ListReq) (res *v1.{{.Entity}}ListRes, err error)
	// Create 新增。
	Create(ctx context.Context, req *v1.{{.Entity}}CreateReq) (res *v1.{{.Entity}}CreateRes, err error)
	// Update 修改。
	Update(ctx context.Context, req *v1.{{.Entity}}UpdateReq) (res *v1.{{.Entity}}UpdateRes, err error)
	// Delete 删除 (软删)。
	Delete(ctx context.Context, req *v1.{{.Entity}}DeleteReq) (res *v1.{{.Entity}}DeleteRes, err error)
}

var local{{.Entity}} I{{.Entity}}

func {{.Entity}}() I{{.Entity}} {
	if local{{.Entity}} == nil {
		panic("implement not found for interface I{{.Entity}}, forgot register?")
	}
	return local{{.Entity}}
}

func Register{{.Entity}}(i I{{.Entity}}) {
	local{{.Entity}} = i
}
`

const logicTmpl = `// Package {{.Mod}} {{.Title}}业务逻辑。 (crudgen 生成)
package {{.Mod}}

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"

	v1 "hinay.cn/admin/api/{{.Mod}}/v1"
	"hinay.cn/admin/internal/dao"
	"hinay.cn/admin/internal/model"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/response"
	"hinay.cn/admin/utility/xerror"
)

type s{{.Entity}} struct{}

func init() {
	service.Register{{.Entity}}(New{{.Entity}}())
}

func New{{.Entity}}() *s{{.Entity}} {
	return &s{{.Entity}}{}
}

// List 分页列表。
func (s *s{{.Entity}}) List(ctx context.Context, req *v1.{{.Entity}}ListReq) (res *v1.{{.Entity}}ListRes, err error) {
	q := dao.{{.Entity}}.Ctx(ctx).Where("deleted_at IS NULL")
	if req.Keyword != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		// 括号分组: 避免 OR 条件逃逸出软删过滤
		conds := make([]string, 0, {{len .StringCols}})
		args := make([]any, 0, {{len .StringCols}})
{{- range .StringCols}}
		conds = append(conds, "{{.Name}} LIKE ?")
		args = append(args, kw)
{{- end}}
		if len(conds) > 0 {
			q = q.Where("("+strings.Join(conds, " OR ")+")", args...)
		}
	}
	total, err := q.Count()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	var rows []*model.{{.Entity}}
	if err = q.Page(req.Page, req.PageSize).Order("id DESC").Scan(&rows); err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err)
	}
	items := make([]*v1.{{.Entity}}Item, 0, len(rows))
	for _, r := range rows {
		items = append(items, &v1.{{.Entity}}Item{
			Id: r.Id,
{{- range .Columns}}
			{{.GoName}}: r.{{.GoName}},
{{- end}}
			CreatedAt: r.CreatedAt,
		})
	}
	page := response.Page(items, int64(total), req.Page, req.PageSize)
	out := v1.{{.Entity}}ListRes(page)
	return &out, nil
}

// Create 新增 (create_id/update_id 由 ormfill 自动填充)。
func (s *s{{.Entity}}) Create(ctx context.Context, req *v1.{{.Entity}}CreateReq) (res *v1.{{.Entity}}CreateRes, err error) {
	_, err = dao.{{.Entity}}.Ctx(ctx).Data(g.Map{
{{- range .FormColumns}}
		"{{.Name}}": req.{{.GoName}},
{{- end}}
	}).Insert()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "新增失败")
	}
	return &v1.{{.Entity}}CreateRes{}, nil
}

// Update 修改。
func (s *s{{.Entity}}) Update(ctx context.Context, req *v1.{{.Entity}}UpdateReq) (res *v1.{{.Entity}}UpdateRes, err error) {
	_, err = dao.{{.Entity}}.Ctx(ctx).Where("id", req.Id).Where("deleted_at IS NULL").
		Data(g.Map{
{{- range .FormColumns}}
			"{{.Name}}": req.{{.GoName}},
{{- end}}
		}).Update()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "修改失败")
	}
	return &v1.{{.Entity}}UpdateRes{}, nil
}

// Delete 删除 (软删)。
func (s *s{{.Entity}}) Delete(ctx context.Context, req *v1.{{.Entity}}DeleteReq) (res *v1.{{.Entity}}DeleteRes, err error) {
	_, err = dao.{{.Entity}}.Ctx(ctx).Where("id", req.Id).Delete()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "删除失败")
	}
	return &v1.{{.Entity}}DeleteRes{}, nil
}
`

const modelTmpl = `package model

import "github.com/gogf/gf/v2/os/gtime"

// {{.Entity}} 数据库实体 ({{.Table}})。 (crudgen 生成)
type {{.Entity}} struct {
	Id        uint64      {{bt}}json:"id"        orm:"id"{{bt}}
{{- range .Columns}}
	{{.GoName}} {{.GoType}} {{bt}}json:"{{camel .GoName}}"  orm:"{{.Name}}"{{bt}}
{{- end}}
	CreatedAt *gtime.Time {{bt}}json:"createdAt" orm:"created_at"{{bt}}
	UpdatedAt *gtime.Time {{bt}}json:"updatedAt" orm:"updated_at"{{bt}}
}
`

const entityTmpl = `// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// {{.Entity}} is the golang structure for table {{.Table}}.
type {{.Entity}} struct {
	Id        uint64      {{bt}}json:"id"        orm:"id"         description:"ID"{{bt}}
{{- range .Columns}}
	{{.GoName}} {{.GoType}} {{bt}}json:"{{camel .GoName}}"  orm:"{{.Name}}"  description:"{{.Comment}}"{{bt}}
{{- end}}
	CreatedAt *gtime.Time {{bt}}json:"createdAt" orm:"created_at" description:"创建时间"{{bt}}
	UpdatedAt *gtime.Time {{bt}}json:"updatedAt" orm:"updated_at" description:"更新时间"{{bt}}
	DeletedAt *gtime.Time {{bt}}json:"deletedAt" orm:"deleted_at" description:"删除时间(软删)"{{bt}}
}
`

const doTmpl = `// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// {{.Entity}} is the golang structure of table {{.Table}} for DAO operations like Where/Data.
type {{.Entity}} struct {
	g.Meta    {{bt}}orm:"table:{{.Table}}, do:true"{{bt}}
	Id        any         // ID
{{- range .Columns}}
	{{.GoName}} any         // {{.Comment}}
{{- end}}
	CreatedAt *gtime.Time // 创建时间
	UpdatedAt *gtime.Time // 更新时间
	DeletedAt *gtime.Time // 删除时间(软删)
}
`

const daoTmpl = `// =================================================================================
// This file is auto-generated by the GoFrame CLI tool. You may modify it as needed.
// =================================================================================

package dao

import (
	"hinay.cn/admin/internal/dao/internal"
)

// {{camel .Entity}}Dao is the data access object for the table {{.Table}}.
type {{camel .Entity}}Dao struct {
	*internal.{{.Entity}}Dao
}

var (
	// {{.Entity}} is a globally accessible object for table {{.Table}} operations.
	{{.Entity}} = {{camel .Entity}}Dao{internal.New{{.Entity}}Dao()}
)

// Add your custom methods and functionality below.
`

const daoInternalTmpl = `// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// {{.Entity}}Dao is the data access object for the table {{.Table}}.
type {{.Entity}}Dao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  {{.Entity}}Columns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// {{.Entity}}Columns defines and stores column names for the table {{.Table}}.
type {{.Entity}}Columns struct {
	Id        string // ID
{{- range .Columns}}
	{{.GoName}} string // {{.Comment}}
{{- end}}
	CreatedAt string // 创建时间
	UpdatedAt string // 更新时间
	DeletedAt string // 删除时间(软删)
}

// {{camel .Entity}}Columns holds the columns for the table {{.Table}}.
var {{camel .Entity}}Columns = {{.Entity}}Columns{
	Id:        "id",
{{- range .Columns}}
	{{.GoName}}: "{{.Name}}",
{{- end}}
	CreatedAt: "created_at",
	UpdatedAt: "updated_at",
	DeletedAt: "deleted_at",
}

// New{{.Entity}}Dao creates and returns a new DAO object for table data access.
func New{{.Entity}}Dao(handlers ...gdb.ModelHandler) *{{.Entity}}Dao {
	return &{{.Entity}}Dao{
		group:    "default",
		table:    "{{.Table}}",
		columns:  {{camel .Entity}}Columns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *{{.Entity}}Dao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *{{.Entity}}Dao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *{{.Entity}}Dao) Columns() {{.Entity}}Columns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *{{.Entity}}Dao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO.
func (dao *{{.Entity}}Dao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
func (dao *{{.Entity}}Dao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
`

const webApiTmpl = `/**
 * {{.Title}} API。 (crudgen 生成)
 */
import { useRequest } from '~/composables/useRequest'

export function use{{.Entity}}Api() {
  const r = useRequest()
  const base = '/{{.Mod}}'
  return {
    list: (params: { keyword?: string; page?: number; pageSize?: number }) =>
      r.get<{ list: any[]; total: number }>(base, params),
    create: (data: any) => r.post(base, data),
    update: (id: number, data: any) => r.put(base + '/' + id, data),
    remove: (id: number) => r.del(base + '/' + id),
  }
}
`

const webPageTmpl = `<script setup lang="ts">
/**
 * {{.Title}} (crudgen 生成)
 */
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Search, Refresh, Plus, Edit, Delete } from '@element-plus/icons-vue'
import { use{{.Entity}}Api } from '~/composables/useApi'

definePageMeta({ title: '{{.Title}}' })
defineOptions({ name: '{{.Mod}}' })

const api = use{{.Entity}}Api()
const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const query = reactive({ keyword: '', page: 1, pageSize: 10 })

const dialogVisible = ref(false)
const submitting = ref(false)
const editingId = ref(0)
const formRef = ref<FormInstance>()
const form = reactive({
{{- range .FormColumns}}
  {{camel .GoName}}: {{if eq .Kind "switch"}}1{{else if eq .Kind "number"}}0{{else}}''{{end}},
{{- end}}
})
const rules: FormRules = {
{{- if .FirstGoName}}
  {{camel .FirstGoName}}: [{ required: true, message: '必填', trigger: 'blur' }],
{{- end}}
}

async function loadList() {
  loading.value = true
  try {
    const res = await api.list({ ...query })
    list.value = res.list || []
    total.value = res.total || 0
  }
  finally {
    loading.value = false
  }
}

function resetQuery() {
  query.keyword = ''
  query.page = 1
  loadList()
}

function openCreate() {
  editingId.value = 0
  Object.assign(form, {
{{- range .FormColumns}}
    {{camel .GoName}}: {{if eq .Kind "switch"}}1{{else if eq .Kind "number"}}0{{else}}''{{end}},
{{- end}}
  })
  dialogVisible.value = true
}

function openEdit(row: any) {
  editingId.value = row.id
  Object.assign(form, {
{{- range .FormColumns}}
    {{camel .GoName}}: row.{{camel .GoName}}{{if eq .Kind "switch"}} ?? 1{{end}},
{{- end}}
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    if (editingId.value) {
      await api.update(editingId.value, { ...form })
      ElMessage.success('修改成功')
    }
    else {
      await api.create({ ...form })
      ElMessage.success('新增成功')
    }
    dialogVisible.value = false
    loadList()
  }
  finally {
    submitting.value = false
  }
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm('确认删除该条记录吗?', '提示', { type: 'warning' })
  await api.remove(row.id)
  ElMessage.success('已删除')
  loadList()
}

onMounted(loadList)
</script>

<template>
  <div class="page">
    <el-card>
      <el-form inline @submit.prevent>
        <el-form-item label="关键词">
          <el-input
            v-model="query.keyword"
            placeholder="模糊搜索"
            clearable
            style="width:200px"
            @keyup.enter="() => { query.page = 1; loadList() }"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="() => { query.page = 1; loadList() }">查询</el-button>
          <el-button :icon="Refresh" @click="resetQuery">重置</el-button>
          <el-button v-permission="'{{.PermPrefix}}:create'" type="success" :icon="Plus" @click="openCreate">新增</el-button>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" border stripe style="margin-top:8px">
        <el-table-column prop="id" label="ID" width="80" />
{{- range .ListColumns}}
        <el-table-column {{if eq .Kind "switch"}}label="{{if .Comment}}{{.Comment}}{{else}}状态{{end}}" width="90"{{else}}prop="{{camel .GoName}}" label="{{if .Comment}}{{.Comment}}{{else}}{{.GoName}}{{end}}" min-width="120"{{end}}>
{{- if eq .Kind "switch"}}
          <template #default="{ row }">
            <el-tag :type="row.{{camel .GoName}} === 1 ? 'success' : 'info'">
              {{printf "{{ row.%s === 1 ? '启用' : '禁用' }}" (camel .GoName)}}
            </el-tag>
          </template>
{{- end}}
        </el-table-column>
{{- end}}
        <el-table-column prop="createdAt" label="创建时间" width="170" />
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'{{.PermPrefix}}:update'" link type="primary" :icon="Edit" @click="openEdit(row)">编辑</el-button>
            <el-button v-permission="'{{.PermPrefix}}:delete'" link type="danger" :icon="Delete" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        style="margin-top:16px;justify-content:flex-end"
        @current-change="loadList"
        @size-change="loadList"
      />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑{{.Title}}' : '新增{{.Title}}'" width="560px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
{{- range .FormColumns}}
{{- if eq .Kind "switch"}}
        <el-form-item label="{{if .Comment}}{{.Comment}}{{else}}状态{{end}}">
          <el-radio-group v-model="form.{{camel .GoName}}">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
{{- else if eq .Kind "textarea"}}
        <el-form-item label="{{if .Comment}}{{.Comment}}{{else}}{{.GoName}}{{end}}" prop="{{camel .GoName}}">
          <el-input v-model="form.{{camel .GoName}}" type="textarea" :rows="3" />
        </el-form-item>
{{- else if eq .Kind "number"}}
        <el-form-item label="{{if .Comment}}{{.Comment}}{{else}}{{.GoName}}{{end}}" prop="{{camel .GoName}}">
          <el-input-number v-model="form.{{camel .GoName}}" :min="0" style="width:100%" />
        </el-form-item>
{{- else}}
        <el-form-item label="{{if .Comment}}{{.Comment}}{{else}}{{.GoName}}{{end}}" prop="{{camel .GoName}}">
          <el-input v-model="form.{{camel .GoName}}" />
        </el-form-item>
{{- end}}
{{- end}}
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page {
  padding: 0;
}
</style>
`

const sqlTmpl = `-- ============================================================
-- {{.Title}} (crudgen 生成: 菜单/按钮/API 种子)
-- 表: {{.Table}}
-- ============================================================

-- 菜单 (顶级, id={{.MenuId}})
INSERT INTO sys_menu (id, parent_id, name, type, path, component, icon, permission, sort, visible, status) VALUES
  ({{.MenuId}}, 0, '{{.Title}}', 2, '/{{.Mod}}', '{{.Mod}}/index', 'Document', '{{.PermPrefix}}:list', 200, 1, 1);

-- 按钮权限
INSERT INTO sys_menu (id, parent_id, name, type, path, component, icon, permission, sort, visible, status) VALUES
  ({{.ButtonBase}}, {{.MenuId}}, '{{.Title}}新增', 3, '', '', '', '{{.PermPrefix}}:create', 1, 0, 1),
  ({{add1 .ButtonBase}}, {{.MenuId}}, '{{.Title}}修改', 3, '', '', '', '{{.PermPrefix}}:update', 2, 0, 1),
  ({{add1 (add1 .ButtonBase)}}, {{.MenuId}}, '{{.Title}}删除', 3, '', '', '', '{{.PermPrefix}}:delete', 3, 0, 1);

-- API 资源
INSERT INTO sys_api (path, method, group_name, description) VALUES
  ('/api/v1/{{.Mod}}',     'GET',    '{{.Title}}', '{{.Title}}列表'),
  ('/api/v1/{{.Mod}}',     'POST',   '{{.Title}}', '新增{{.Title}}'),
  ('/api/v1/{{.Mod}}/:id', 'PUT',    '{{.Title}}', '修改{{.Title}}'),
  ('/api/v1/{{.Mod}}/:id', 'DELETE', '{{.Title}}', '删除{{.Title}}');

-- Casbin: 内置超管角色(id=1) 不种 p 策略 (API 中间件全放行 + 菜单全量下发),
-- 新菜单默认仅超管可见, 其他角色在「角色管理-分配权限」中按需勾选。
`

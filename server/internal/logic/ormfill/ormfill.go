// Package ormfill ORM 审计字段自动填充 (create_id / update_id)。
//
// 实现方式: 参照 GoFrame "接口回调" 机制 (goframe.org/docs/core/gdb-interface-callback),
// 继承 mysql 驱动并重写 DoInsert / DoUpdate, 在 init() 中以 "mysql" 名重新注册覆盖内置驱动,
// 配置文件无需任何改动。由于不触碰 internal/dao 下的任何文件, `gf gen dao` 重新生成后无需恢复。
//
// 填充规则 (仅对 fillTables 白名单内的业务表生效):
//   - INSERT: create_id = 当前登录用户, update_id = 当前登录用户;
//   - UPDATE: update_id = 当前登录用户;
//   - 无登录上下文 (定时任务/异步日志等后台写入) 时不填充, 字段保持默认值 0。
//
// 业务代码不得手动写入这两个字段; 新业务表需在 fillTables 登记表名
// (白名单必须与实际建了这两列的表严格一致, 多填会报未知列)。
package ormfill

import (
	"context"
	"database/sql"
	"strings"

	"github.com/gogf/gf/contrib/drivers/mysql/v2"
	"github.com/gogf/gf/v2/database/gdb"

	"hinay.cn/admin/internal/consts"
	"hinay.cn/admin/internal/model"
)

// 审计字段列名。
const (
	columnCreateId = "create_id"
	columnUpdateId = "update_id"
)

// fillTables 需要 create_id/update_id 审计字段自动填充的业务表白名单。
// 注意: 必须与实际建了这两列的表严格一致 (白名单外不填充, 多填会报未知列)。
// 说明: biz_message_read / biz_message_target 为无 created_at 的纯关联表
// (已读关系的 user_id 即行为人), 不纳入审计字段。
var fillTables = map[string]struct{}{
	"sys_user":      {},
	"sys_role":      {},
	"sys_role_org":  {},
	"sys_org":       {},
	"sys_menu":      {},
	"sys_api":       {},
	"sys_dict_type": {},
	"sys_dict_data": {},
	"sys_config":    {},
	"sys_file":      {},
	"sys_job":       {},
	"biz_message":   {},
}

// Driver 审计字段填充驱动: 继承内置 mysql 驱动, 重写 DoInsert / DoUpdate。
type Driver struct {
	*mysql.Driver
}

func init() {
	// 覆盖内置 "mysql" 驱动 (gdb.Register 为覆盖式注册):
	// 本包 import 了 contrib mysql, 其 init 先于本包执行, 因此这里的注册必然后到后赢;
	// logic 包的 init 在任何数据库实例创建之前运行, 覆盖对存量连接无竞态影响。
	if err := gdb.Register("mysql", &Driver{}); err != nil {
		panic(err)
	}
}

// New 组装驱动实例 (接口回调约定的驱动构造入口)。
func (d *Driver) New(core *gdb.Core, node *gdb.ConfigNode) (gdb.DB, error) {
	return &Driver{&mysql.Driver{Core: core}}, nil
}

// DoInsert INSERT 回调: 为白名单表补 create_id / update_id。
func (d *Driver) DoInsert(ctx context.Context, link gdb.Link, table string, list gdb.List, option gdb.DoInsertOption) (result sql.Result, err error) {
	if _, ok := fillTables[normalizeTable(table)]; ok {
		if uid := loginUserId(ctx); uid > 0 {
			for _, record := range list {
				record[columnCreateId] = uid
				record[columnUpdateId] = uid
			}
		}
	}
	return d.Driver.DoInsert(ctx, link, table, list, option)
}

// DoUpdate UPDATE 回调: 为白名单表补 update_id (data 为字符串更新语句时跳过)。
func (d *Driver) DoUpdate(ctx context.Context, link gdb.Link, table string, data any, condition string, args ...any) (result sql.Result, err error) {
	if _, ok := fillTables[normalizeTable(table)]; ok {
		if uid := loginUserId(ctx); uid > 0 {
			if m, ok := data.(map[string]any); ok {
				m[columnUpdateId] = uid
			}
		}
	}
	return d.Driver.DoUpdate(ctx, link, table, data, condition, args...)
}

// normalizeTable 还原驱动层收到的表名为裸表名。
// Model 创建时表名已经过 QuotePrefixTableName 处理 (gdb_model.go:130),
// 到达驱动层时形如 "`sys_config`" 或 "`schema`.`sys_config`"; 带 " AS " 别名时取别名前的主表。
// 白名单登记的是裸表名, 不做这步还原会永久 miss。
func normalizeTable(table string) string {
	t := strings.ReplaceAll(table, "`", "")
	if i := strings.Index(strings.ToUpper(t), " AS "); i >= 0 {
		t = t[:i]
	}
	if i := strings.LastIndex(t, "."); i >= 0 {
		t = t[i+1:]
	}
	return t
}

// loginUserId 从请求上下文提取当前登录用户 ID (与 Auth 中间件的
// context.WithValue(ctx, consts.CtxUserKey, user) 双写键一致),
// 无登录上下文 (后台任务/异步写入) 返回 0。
func loginUserId(ctx context.Context) uint64 {
	if v := ctx.Value(consts.CtxUserKey); v != nil {
		if u, ok := v.(*model.LoginUser); ok && u != nil {
			return u.UserId
		}
	}
	return 0
}

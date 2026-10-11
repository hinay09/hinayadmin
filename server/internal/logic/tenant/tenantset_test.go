// tenantset_test.go —— 应用授权集闭包/差集/平台专属判定纯函数测试 (docs/multi-tenant.md §7.1)。
package tenant

import (
	"testing"

	"hinay.cn/admin/internal/consts"
)

// demoMenus 精简菜单骨架 (id/parent/type), 形态对齐 init.sql 种子。
func demoMenus() []MenuLite {
	return []MenuLite{
		{Id: 1, ParentId: 0, Type: 1},    // 系统管理 目录
		{Id: 10, ParentId: 1, Type: 2},   // 用户管理
		{Id: 101, ParentId: 10, Type: 3}, // 用户新增按钮
		{Id: 11, ParentId: 1, Type: 2},   // 角色管理
		{Id: 2, ParentId: 0, Type: 1},    // 系统监控 目录 (平台专属)
		{Id: 22, ParentId: 2, Type: 2},   // 服务器监控 (平台专属)
		{Id: 3, ParentId: 0, Type: 1},    // 系统工具 目录 (平台专属)
		{Id: 50, ParentId: 0, Type: 1},   // 消息中心
		{Id: 51, ParentId: 50, Type: 2},
		{Id: 511, ParentId: 51, Type: 3},
		{Id: 9000, ParentId: 0, Type: 1}, // 审批中心
		{Id: 9010, ParentId: 9000, Type: 2},
		{Id: consts.MenuTenantManage, ParentId: 1, Type: 2}, // 租户管理 (平台专属)
	}
}

func TestClosureOf(t *testing.T) {
	menus := demoMenus()
	// 授权 10 (用户管理): 闭包 = 10 + 按钮 101 + 祖先目录 1
	got := closureOf(menus, map[uint64]struct{}{10: {}})
	want := map[uint64]struct{}{1: {}, 10: {}, 101: {}}
	if len(got) != len(want) {
		t.Fatalf("closure len = %d, want %d (%v)", len(got), len(want), got)
	}
	for id := range want {
		if _, ok := got[id]; !ok {
			t.Fatalf("closure missing %d", id)
		}
	}
	// 按钮直接授权: 随父闭包 (按钮本身合法进入闭包, 写入侧由 validateGrantSet 拦截按钮级授权行)
	got = closureOf(menus, map[uint64]struct{}{511: {}})
	if _, ok := got[511]; !ok {
		t.Fatal("按钮应在闭包内")
	}
	if _, ok := got[50]; !ok {
		t.Fatal("祖先目录 50 应补全 (树渲染完整性)")
	}
	// 不存在的菜单 id 被忽略
	got = closureOf(menus, map[uint64]struct{}{99999: {}})
	if len(got) != 0 {
		t.Fatalf("不存在的菜单应被忽略, got %v", got)
	}
	// 空授权集 → 空闭包
	if got := closureOf(menus, nil); len(got) != 0 {
		t.Fatalf("空授权集应返回空闭包")
	}
}

func TestDiffSets(t *testing.T) {
	old := map[uint64]struct{}{10: {}, 11: {}, 50: {}}
	new := map[uint64]struct{}{11: {}, 9000: {}}
	added, removed := diffSets(old, new)
	if _, ok := added[9000]; !ok || len(added) != 1 {
		t.Fatalf("added = %v, want {9000}", added)
	}
	if len(removed) != 2 {
		t.Fatalf("removed = %v, want {10,50}", removed)
	}
	if _, ok := removed[10]; !ok {
		t.Fatalf("removed 缺少 10: %v", removed)
	}
	if _, ok := removed[50]; !ok {
		t.Fatalf("removed 缺少 50: %v", removed)
	}
}

func TestIsPlatformExclusive(t *testing.T) {
	menus := demoMenus()
	cases := []struct {
		id   uint64
		want bool
		desc string
	}{
		{22, true, "监控子树页面"},
		{2, true, "监控目录本身"},
		{3, true, "工具目录"},
		{consts.MenuTenantManage, true, "租户管理"},
		{10, false, "用户管理可授权"},
		{9000, false, "审批中心可授权"},
		{1, false, "系统管理目录可授权 (子集里有专属页但目录本身可)"},
	}
	for _, c := range cases {
		if got := isPlatformExclusive(menus, c.id); got != c.want {
			t.Errorf("isPlatformExclusive(%d) = %v, want %v (%s)", c.id, got, c.want, c.desc)
		}
	}
}

func TestBaseMenuClosure(t *testing.T) {
	b := baseMenuClosure()
	if _, ok := b[consts.MenuDashboard]; !ok {
		t.Fatal("仪表盘应恒授")
	}
	if _, ok := b[consts.MenuProfile]; !ok {
		t.Fatal("个人中心应恒授")
	}
}

func TestValidateGrantSet(t *testing.T) {
	menus := demoMenus()
	s := &sTenant{}
	// 正常目录/页面级
	if err := s.validateGrantSet(menus, map[uint64]struct{}{10: {}, 9000: {}}); err != nil {
		t.Fatalf("合法授权集被拒: %v", err)
	}
	// 按钮级拒绝
	if err := s.validateGrantSet(menus, map[uint64]struct{}{101: {}}); err == nil {
		t.Fatal("按钮级授权应被拒 (按钮随父页面闭包)")
	}
	// 平台专属拒绝
	if err := s.validateGrantSet(menus, map[uint64]struct{}{22: {}}); err == nil {
		t.Fatal("平台专属菜单应被拒")
	}
	if err := s.validateGrantSet(menus, map[uint64]struct{}{consts.MenuTenantManage: {}}); err == nil {
		t.Fatal("租户管理菜单应被拒")
	}
	// 不存在拒绝
	if err := s.validateGrantSet(menus, map[uint64]struct{}{424242: {}}); err == nil {
		t.Fatal("不存在的菜单应被拒")
	}
}

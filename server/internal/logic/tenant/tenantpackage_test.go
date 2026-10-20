// tenantpackage_test.go —— 套餐 menu_ids JSON 编解码/集合工具纯函数测试 (docs/multi-tenant.md §7.2)。
package tenant

import (
	"reflect"
	"testing"

	"hinay.cn/admin/internal/consts"
)

func TestEncodeParseMenuIds(t *testing.T) {
	cases := []struct {
		name string
		in   []uint64
	}{
		{"常规", []uint64{30, 40, 51}},
		{"乱序", []uint64{51, 10, 40}},
		{"单元素", []uint64{9350}},
		{"空", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseMenuIds(encodeMenuIds(c.in))
			want := c.in
			if want == nil {
				want = []uint64{}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("round-trip 失败: got %v want %v", got, want)
			}
		})
	}
}

func TestParseMenuIdsDirtyInput(t *testing.T) {
	// 非法 JSON / 空串 / 0 元素均安全降级 (nil / 剔除 0), 不 panic
	if got := parseMenuIds("not-json"); got != nil {
		t.Fatalf("非法 JSON 应返回 nil, got %v", got)
	}
	if got := parseMenuIds(""); got != nil {
		t.Fatalf("空串应返回 nil, got %v", got)
	}
	if got := parseMenuIds("[0,30,40]"); !reflect.DeepEqual(got, []uint64{30, 40}) {
		t.Fatalf("0 元素应被剔除, got %v", got)
	}
}

func TestEncodeMenuSetSorted(t *testing.T) {
	set := map[uint64]struct{}{52: {}, 30: {}, 40: {}}
	if got := encodeMenuSet(set); got != "[30,40,52]" {
		t.Fatalf("集合编码应升序稳定: got %s", got)
	}
	if got := encodeMenuSet(map[uint64]struct{}{}); got != "[]" {
		t.Fatalf("空集编码应为 [], got %s", got)
	}
}

func TestSortedIds(t *testing.T) {
	if got := sortedIds(map[uint64]struct{}{9350: {}, 10: {}, 1: {}}); !reflect.DeepEqual(got, []uint64{1, 10, 9350}) {
		t.Fatalf("sortedIds 应升序: got %v", got)
	}
}

// 平台专属判定须覆盖套餐管理页 (9360), 防止套餐模板把平台入口授予租户。
func TestPackagePagePlatformExclusive(t *testing.T) {
	menus := append(demoMenus(), MenuLite{Id: consts.MenuTenantPackage, ParentId: 1, Type: 2})
	if !isPlatformExclusive(menus, consts.MenuTenantPackage) {
		t.Fatal("租户套餐页 (9360) 应为平台专属, 不可进任何租户授权集/套餐模板")
	}
}

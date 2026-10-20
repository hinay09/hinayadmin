// tenantset.go —— 应用授权集的纯函数计算 (闭包/差集/平台专属判定), 表驱动单测覆盖。
package tenant

import "hinay.cn/admin/internal/consts"

// MenuLite 菜单骨架 (闭包计算仅需 id/parent/type)。
type MenuLite struct {
	Id       uint64
	ParentId uint64
	Type     int
}

// platformExclusiveMenus 平台专属页面 (docs/multi-tenant.md §7.1/§7.2):
// 系统监控(2)/系统工具(3) 目录子树 + 平台目录维护页 (12 菜单/13 API/14 字典/17 全局配置)
// + 租户管理(9300) 子树 + 租户套餐(9360) 子树 —— 永不进任何租户授权集。
var platformExclusiveMenus = map[uint64]struct{}{
	2: {}, 3: {}, 12: {}, 13: {}, 14: {}, 17: {},
	consts.MenuTenantManage: {}, consts.MenuTenantPackage: {},
}

// closureOf 计算授权集的有效闭包: 授权行 ∪ 子孙(按钮随父页面闭包) ∪ 祖先(菜单树完整性)。
func closureOf(menus []MenuLite, granted map[uint64]struct{}) map[uint64]struct{} {
	if len(granted) == 0 {
		return map[uint64]struct{}{}
	}
	byId := make(map[uint64]MenuLite, len(menus))
	children := make(map[uint64][]uint64, len(menus))
	for _, m := range menus {
		byId[m.Id] = m
		children[m.ParentId] = append(children[m.ParentId], m.Id)
	}
	out := make(map[uint64]struct{}, len(granted)*2)
	// 向下闭包 (按钮/子页面)
	var addDesc func(id uint64)
	addDesc = func(id uint64) {
		if _, ok := out[id]; ok {
			return
		}
		out[id] = struct{}{}
		for _, c := range children[id] {
			addDesc(c)
		}
	}
	for id := range granted {
		if _, ok := byId[id]; ok { // 授权行必须真实存在
			addDesc(id)
		}
	}
	// 向上补祖先 (树渲染完整性)
	for id := range out {
		p, ok := byId[id]
		if !ok {
			continue
		}
		for p.ParentId != 0 {
			if _, ok := byId[p.ParentId]; !ok {
				break
			}
			out[p.ParentId] = struct{}{}
			p = byId[p.ParentId]
		}
	}
	return out
}

// diffSets 集合差: 返回 added (new-old) 与 removed (old-new)。
func diffSets(oldSet, newSet map[uint64]struct{}) (added, removed map[uint64]struct{}) {
	added = make(map[uint64]struct{})
	removed = make(map[uint64]struct{})
	for id := range newSet {
		if _, ok := oldSet[id]; !ok {
			added[id] = struct{}{}
		}
	}
	for id := range oldSet {
		if _, ok := newSet[id]; !ok {
			removed[id] = struct{}{}
		}
	}
	return added, removed
}

// isPlatformExclusive 判定菜单 (含祖先链) 是否平台专属。
func isPlatformExclusive(menus []MenuLite, id uint64) bool {
	byId := make(map[uint64]MenuLite, len(menus))
	for _, m := range menus {
		byId[m.Id] = m
	}
	cur, ok := byId[id]
	for ok {
		if _, ex := platformExclusiveMenus[cur.Id]; ex {
			return true
		}
		if cur.ParentId == 0 {
			return false
		}
		cur, ok = byId[cur.ParentId]
	}
	return false
}

// baseMenuClosure 基础菜单恒授 (30 仪表盘 / 40 个人中心, 代码常量, 不依赖授权集)。
func baseMenuClosure() map[uint64]struct{} {
	return map[uint64]struct{}{
		consts.MenuDashboard: {},
		consts.MenuProfile:   {},
	}
}

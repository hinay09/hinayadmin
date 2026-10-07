// Package flow 自由审批流。
package flow

import (
	"encoding/json"
	"fmt"
)

// ============================================================
// 状态常量 (与 DDL 注释对齐)
// ============================================================

// 定义状态。
const (
	defStatusDraft     = 0 // 草稿
	defStatusPublished = 1 // 已发布
	defStatusDisabled  = 2 // 已停用
)

// 实例状态。
const (
	instStatusRunning    = 1 // 运行中
	instStatusApproved   = 2 // 已通过
	instStatusRejected   = 3 // 已废弃 (历史"整单驳回"; 现驳回退回为 6, 不再使用)
	instStatusCanceled   = 4 // 已撤销 (终态)
	instStatusTerminated = 5 // 已终止
	instStatusReturned   = 6 // 已退回 (被驳回, 待发起人修改后重提或撤销)
	instStatusWithdrawn  = 7 // 已撤回 (发起人主动收回, 待修改后重提或撤销; 区别于撤销终态 4)
)

// 任务节点类型。
const (
	taskNodeTypeApprove = 1 // 审批
	taskNodeTypeCC      = 2 // 抄送
)

// 签核方式。
const (
	taskSignAny = 1 // 或签: 任一人处理即过
	taskSignAll = 2 // 会签: 全部同意才过
)

// 任务状态。
const (
	taskStatusPending       = 1 // 待办/待阅
	taskStatusApproved      = 2 // 已同意/已阅
	taskStatusRejected      = 3 // 已驳回
	taskStatusTransferred   = 4 // 已转出
	taskStatusVoid          = 5 // 已作废(或签被抢先/流程终止连带)
	taskStatusInvalid       = 6 // 已失效(退回/撤销后原审批同意不再计入当前轮, 历史记录保留)
	taskStatusDelegated     = 7 // 已委派(原任务挂起, 等待被委托人代办后回到本任务终审)
	taskStatusDelegatedDone = 8 // 委办完成(被委托人已提交意见, 已回到原审批人; 不计入会签/或签统计)
)

// 审批超时策略 (approver 节点 timeoutAction 配置)。
const (
	timeoutActionRemind   = "remind"   // 逾期提醒: 通知审批人 (默认, 每任务一次)
	timeoutActionTransfer = "transfer" // 自动转办: 系统转给指定人 (转办后不再计时, 防循环)
	timeoutActionApprove  = "approve"  // 自动通过: 系统代为同意并推进
)

// 节点类型标识。
const (
	nodeTypeStart     = "start"
	nodeTypeApprover  = "approver"
	nodeTypeCC        = "cc"
	nodeTypeCondition = "condition"
)

// 审批人解析方式。
const (
	approverTypeUser       = "user"       // 指定成员
	approverTypeRole       = "role"       // 指定角色 (运行时展开为角色下用户)
	approverTypePost       = "post"       // 指定岗位 (岗位管理, 运行时展开为挂岗用户)
	approverTypeSelfSelect = "selfSelect" // 发起人自选 (发起时提交)
	approverTypeSuperior   = "superior"   // 部门主管 (发起人组织起逐级向上找挂主管岗者)
	approverTypeInitiator  = "initiator"  // 发起人本人
)

// ============================================================
// 流程/表单配置模型 (flow_conf / form_conf 的 Go 结构)
// ============================================================

// FormField 表单字段定义。
type FormField struct {
	Key      string   `json:"key"`   // 字段标识 (表单数据键)
	Label    string   `json:"label"` // 显示名
	Type     string   `json:"type"`  // input/number/textarea/select/date
	Options  []string `json:"options,omitempty"`
	Required bool     `json:"required"` // 是否必填
}

// Cond 单个条件: 表单字段 op 值。
type Cond struct {
	Field string `json:"field"`
	Op    string `json:"op"` // eq/ne/gt/lt/ge/le/in
	Value any    `json:"value"`
}

// Branch 条件分支: Conditions 外层 OR、内层 AND。
type Branch struct {
	Id         string   `json:"id"`
	Name       string   `json:"name"`
	IsDefault  bool     `json:"isDefault"`
	Conditions [][]Cond `json:"conditions"`
	Child      *Node    `json:"child"`
}

// Node 流程节点 (纵向链: Child 指向下一节点; 条件节点按分支深入)。
type Node struct {
	Id           string    `json:"id"`
	Type         string    `json:"type"` // start/approver/cc/condition
	Name         string    `json:"name"`
	Child        *Node     `json:"child,omitempty"`
	ApproverType string    `json:"approverType,omitempty"` // approver/cc: user/role/selfSelect/superior/initiator
	ApproverIds  []uint64  `json:"approverIds,omitempty"`
	SignType     string    `json:"signType,omitempty"` // approver: any=或签, all=会签
	Branches     []*Branch `json:"branches,omitempty"` // condition

	// approver: 超时处理 (任务生成时物化为 wf_task.due_time, 由定时任务 flow.timeoutScan 扫描)
	TimeoutHours    int    `json:"timeoutHours,omitempty"`    // 办理期限(小时), 0=不限
	TimeoutAction   string `json:"timeoutAction,omitempty"`   // 超时策略: remind(默认)/transfer/approve
	TimeoutTransfer uint64 `json:"timeoutTransfer,omitempty"` // 自动转办目标用户ID (timeoutAction=transfer 必填)
}

// ============================================================
// 配置解析与校验
// ============================================================

// defaultFormConf 新建定义的默认表单配置。
func defaultFormConf() string {
	b, _ := json.Marshal([]FormField{{
		Key: "reason", Label: "申请事由", Type: "textarea", Required: true,
	}})
	return string(b)
}

// defaultFlowConf 新建定义的默认节点树: 发起人 → 审批人(发起人自选, 或签)。
func defaultFlowConf() string {
	b, _ := json.Marshal(&Node{
		Id: "start", Type: nodeTypeStart, Name: "发起人",
		Child: &Node{
			Id: "n1", Type: nodeTypeApprover, Name: "审批人",
			ApproverType: approverTypeSelfSelect, SignType: "any",
		},
	})
	return string(b)
}

// parseFormConf 解析表单配置。
func parseFormConf(s string) (fields []FormField, err error) {
	if s == "" {
		return nil, nil
	}
	if err = json.Unmarshal([]byte(s), &fields); err != nil {
		return nil, fmt.Errorf("表单配置不是合法 JSON: %w", err)
	}
	return fields, nil
}

// parseFlowConf 解析节点树配置。
func parseFlowConf(s string) (root *Node, err error) {
	if s == "" {
		return nil, fmt.Errorf("节点树配置为空")
	}
	if err = json.Unmarshal([]byte(s), &root); err != nil {
		return nil, fmt.Errorf("节点树配置不是合法 JSON: %w", err)
	}
	return root, nil
}

// validateConf 保存/发布前的结构校验 (表单字段 + 节点树)。
func validateConf(formConf, flowConf string) error {
	fields, err := parseFormConf(formConf)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, f := range fields {
		if f.Key == "" || f.Label == "" {
			return fmt.Errorf("表单字段缺少 key/label")
		}
		if seen[f.Key] {
			return fmt.Errorf("表单字段标识重复: %s", f.Key)
		}
		seen[f.Key] = true
	}
	root, err := parseFlowConf(flowConf)
	if err != nil {
		return err
	}
	if root == nil || root.Type != nodeTypeStart {
		return fmt.Errorf("节点树必须以 start 发起人节点开头")
	}
	ids := map[string]bool{}
	return walkValidate(root, ids)
}

// walkValidate 校验节点链/分支: 节点 ID 唯一、类型与审批人配置完整。
func walkValidate(n *Node, ids map[string]bool) error {
	for cur := n; cur != nil; cur = cur.Child {
		if cur.Id == "" {
			return fmt.Errorf("存在缺少 id 的节点")
		}
		if ids[cur.Id] {
			return fmt.Errorf("节点 id 重复: %s", cur.Id)
		}
		ids[cur.Id] = true
		switch cur.Type {
		case nodeTypeApprover, nodeTypeCC:
			if cur.Name == "" {
				return fmt.Errorf("节点 %s 缺少名称", cur.Id)
			}
			switch cur.ApproverType {
			case approverTypeUser, approverTypeRole, approverTypePost:
				if len(cur.ApproverIds) == 0 {
					return fmt.Errorf("节点「%s」未选择审批/抄送对象", cur.Name)
				}
			case approverTypeSelfSelect:
				if cur.Type == nodeTypeCC {
					return fmt.Errorf("抄送节点「%s」不支持发起人自选", cur.Name)
				}
			case approverTypeSuperior, approverTypeInitiator:
				// 无需配置
			default:
				return fmt.Errorf("节点「%s」未设置审批人规则", cur.Name)
			}
			if cur.Type == nodeTypeApprover && cur.SignType == "" {
				return fmt.Errorf("审批节点「%s」未设置签核方式", cur.Name)
			}
			// 超时配置校验 (仅审批节点; 配错发布即拦, 避免扫描期才发现转办目标缺失)
			if cur.Type == nodeTypeApprover && cur.TimeoutHours != 0 {
				if cur.TimeoutHours < 0 {
					return fmt.Errorf("审批节点「%s」办理期限不能为负数", cur.Name)
				}
				switch cur.TimeoutAction {
				case "", timeoutActionRemind:
					// 未配置按提醒, 合法
				case timeoutActionTransfer:
					if cur.TimeoutTransfer == 0 {
						return fmt.Errorf("审批节点「%s」超时策略为自动转办, 未选择转办对象", cur.Name)
					}
				case timeoutActionApprove:
					// 自动通过无附加配置
				default:
					return fmt.Errorf("审批节点「%s」超时策略不合法: %s", cur.Name, cur.TimeoutAction)
				}
			}
		case nodeTypeCondition:
			if len(cur.Branches) == 0 {
				return fmt.Errorf("条件节点「%s」缺少分支", cur.Name)
			}
			for _, br := range cur.Branches {
				if !br.IsDefault && len(br.Conditions) == 0 {
					return fmt.Errorf("条件分支「%s」未配置条件且非默认分支", br.Name)
				}
				for _, group := range br.Conditions {
					for _, c := range group {
						if c.Field == "" || c.Op == "" {
							return fmt.Errorf("条件分支「%s」存在不完整的条件", br.Name)
						}
					}
				}
				if br.Child != nil {
					if err := walkValidate(br.Child, ids); err != nil {
						return err
					}
				}
			}
		case nodeTypeStart:
			// 发起人节点无额外配置
		default:
			return fmt.Errorf("节点「%s」类型不合法: %s", cur.Name, cur.Type)
		}
	}
	return nil
}

// findNode 在节点树中按 ID 查找节点 (主链 + 分支子链)。
func findNode(root *Node, id string) *Node {
	for cur := root; cur != nil; cur = cur.Child {
		if cur.Id == id {
			return cur
		}
		if cur.Type == nodeTypeCondition {
			for _, br := range cur.Branches {
				if br.Child == nil {
					continue
				}
				if hit := findNode(br.Child, id); hit != nil {
					return hit
				}
			}
		}
	}
	return nil
}

// ============================================================
// 业务回调注册 (业务模块按 flow_key 挂接流程结束回调)
// ============================================================

// BizListener 流程状态回调 (事务提交后尽力调用)。
type BizListener struct {
	OnApproved   func(instanceId, bizId uint64) // 全部节点通过
	OnReturned   func(instanceId, bizId uint64) // 被驳回退回发起人 (可重提)
	OnWithdrawn  func(instanceId, bizId uint64) // 发起人撤回 (可重提); 未注册时回退 OnReturned
	OnCanceled   func(instanceId, bizId uint64) // 发起人撤销 (终态)
	OnTerminated func(instanceId, bizId uint64) // 管理员终止
}

var bizListeners = map[string]BizListener{}

// RegisterBizListener 注册业务回调 (业务包 init() 中调用)。
func RegisterBizListener(flowKey string, l BizListener) {
	bizListeners[flowKey] = l
}

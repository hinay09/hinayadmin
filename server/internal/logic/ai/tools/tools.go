// Package tools AI 助手的服务端工具集 (供 react agent 调用)。
//
// 约定:
//   - 每个工具一个文件: typed 入参/出参结构体 + 执行函数 (utils.InferTool 反射出参数
//     schema, 参数解码/结果编码由框架处理);
//   - 本文件 Set() 是唯一注册表, 新增工具 = 新建一个文件 + 在此追加一项;
//   - 工具只做"定义", 运行时关切 (SSE 步骤事件/限时执行/错误转结果回填/步骤落库)
//     由父包 ai 的 stepTool 装饰器统一包裹 (见 ai/decorate.go), 本包不感知;
//   - 涉及业务数据的工具必须套数据权限 (对齐对应管理页的查询口径), AI 工具不构成越权通道;
//   - 工具在请求上下文中执行 (带登录态), 未登录场景不注册本包工具。
package tools

import (
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// Set 构造全部工具 (顺序稳定; 每次对话调用一次, 由调用方再做装饰)。
func Set() ([]tool.InvokableTool, error) {
	currentTime, err := utils.InferTool("get_current_time",
		"获取服务器当前日期时间 (含星期、时区与 Unix 时间戳)。当用户询问现在几点、今天日期、星期几, 或回答需要依赖当前时间时调用。",
		runCurrentTime)
	if err != nil {
		return nil, err
	}
	userList, err := utils.InferTool("get_user_list",
		"查询系统人员列表 (仅启用账号, 含编码/姓名/电话/邮箱/所属组织机构)。当用户询问某人的联系方式、某部门有哪些人、或需要按姓名/账号/手机号/邮箱找人时调用。结果自动按当前用户的数据权限过滤, 无权限的人员不会出现。",
		runUserList)
	if err != nil {
		return nil, err
	}
	return []tool.InvokableTool{currentTime, userList}, nil
}

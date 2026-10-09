// get_current_time: 服务器当前时间 (可选 IANA 时区)。
package tools

import (
	"context"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata" // 内嵌 IANA 时区库, 保证精简容器里 LoadLocation("Asia/Shanghai") 也可用
)

// currentTimeArgs 入参 (InferTool 据此反射出模型可见的参数 schema)。
type currentTimeArgs struct {
	Timezone string `json:"timezone,omitempty" jsonschema_description:"IANA 时区名, 如 Asia/Shanghai、UTC; 缺省用服务器本地时区"`
}

// currentTimeResult 返回值 (InferTool 序列化为 JSON 结果文本)。
type currentTimeResult struct {
	Timezone string `json:"timezone"`
	Datetime string `json:"datetime"`
	Weekday  string `json:"weekday"`
	Unix     int64  `json:"unix"`
}

// runCurrentTime 返回当前时间; 可选 timezone 指定时区 (IANA 名, 解析失败返回错误,
// 由父包装饰器转为错误结果回填给模型)。
func runCurrentTime(_ context.Context, args currentTimeArgs) (*currentTimeResult, error) {
	loc := time.Local
	if tz := strings.TrimSpace(args.Timezone); tz != "" {
		l, err := time.LoadLocation(tz)
		if err != nil {
			return nil, fmt.Errorf("无法识别的时区 %q, 请使用 IANA 名称 (如 Asia/Shanghai、UTC)", tz)
		}
		loc = l
	}
	now := time.Now().In(loc)
	return &currentTimeResult{
		Timezone: now.Location().String(),
		Datetime: now.Format("2006-01-02 15:04:05"),
		Weekday:  weekdays[now.Weekday()],
		Unix:     now.Unix(),
	}, nil
}

// weekdays 星期中文映射 (time.Weekday: 周日=0)。
var weekdays = [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

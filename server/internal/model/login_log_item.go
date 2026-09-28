// Package model 登录日志 VO。
package model

import "github.com/gogf/gf/v2/os/gtime"

// LoginLogItem 登录日志列表 VO。
type LoginLogItem struct {
	Id        uint64      `json:"id"`
	UserId    uint64      `json:"userId"`
	Username  string      `json:"username"`
	Status    int         `json:"status"`
	Message   string      `json:"message"`
	Ip        string      `json:"ip"`
	UserAgent string      `json:"userAgent"`
	CreatedAt *gtime.Time `json:"createdAt"`
}

// LoginLogEntry 登录日志写入入参。
type LoginLogEntry struct {
	UserId    uint64 // 用户ID(登录用户不存在时为0)
	Username  string
	Status    int    // 1=成功 0=失败
	Message   string // 失败原因
	Ip        string
	UserAgent string
}

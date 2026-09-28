// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

// Package service 代码生成服务接口。
package service

import (
	"context"

	v1 "hinay.cn/admin/api/system/v1"
)

type IGencode interface {
	// Tables 可生成表清单 (information_schema)。
	Tables(ctx context.Context, req *v1.GencodeTablesReq) (res *v1.GencodeTablesRes, err error)
	// Columns 表列信息 + 默认勾选。
	Columns(ctx context.Context, req *v1.GencodeColumnsReq) (res *v1.GencodeColumnsRes, err error)
	// Preview 预览全部生成文件。
	Preview(ctx context.Context, req *v1.GencodePreviewReq) (res *v1.GencodePreviewRes, err error)
	// Download 打包 zip 写入响应流。
	Download(ctx context.Context, req *v1.GencodeDownloadReq) (res *v1.GencodeDownloadRes, err error)
	// Write 直写源码树 (开发环境专用)。
	Write(ctx context.Context, req *v1.GencodeWriteReq) (res *v1.GencodeWriteRes, err error)
}

var localGencode IGencode

func Gencode() IGencode {
	if localGencode == nil {
		panic("implement not found for interface IGencode, forgot register?")
	}
	return localGencode
}

func RegisterGencode(i IGencode) {
	localGencode = i
}

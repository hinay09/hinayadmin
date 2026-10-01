// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	v1 "hinay.cn/admin/api/system/v1"
)

type (
	IGencode interface {
		// Tables 当前库的全部业务表。
		Tables(ctx context.Context, _ *v1.GencodeTablesReq) (res *v1.GencodeTablesRes, err error)
		// Columns 表列信息 + 默认勾选 (约定列标记 skip)。
		Columns(ctx context.Context, req *v1.GencodeColumnsReq) (res *v1.GencodeColumnsRes, err error)
		// Preview 预览全部生成文件内容。
		Preview(ctx context.Context, req *v1.GencodePreviewReq) (res *v1.GencodePreviewRes, err error)
		// Download 打包 zip 写入响应流 (附件下载)。
		Download(ctx context.Context, req *v1.GencodeDownloadReq) (res *v1.GencodeDownloadRes, err error)
		// Write 直写源码树 (仅开发环境: 需检测到 server 源码根)。
		Write(ctx context.Context, req *v1.GencodeWriteReq) (res *v1.GencodeWriteRes, err error)
	}
)

var (
	localGencode IGencode
)

func Gencode() IGencode {
	if localGencode == nil {
		panic("implement not found for interface IGencode, forgot register?")
	}
	return localGencode
}

func RegisterGencode(i IGencode) {
	localGencode = i
}

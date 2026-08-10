package system

import (
	"context"

	"github.com/gogf/gf/v2/net/ghttp"

	"hinay.cn/admin/api/system/v1"
	"hinay.cn/admin/internal/service"
	"hinay.cn/admin/utility/xerror"
)

// FileUpload 上传文件。
func (c *ControllerV1) FileUpload(ctx context.Context, req *v1.FileUploadReq) (res *v1.FileUploadRes, err error) {
	r := ghttp.RequestFromCtx(ctx)
	if r == nil {
		return nil, xerror.New(xerror.CodeBusinessError, "无法获取请求上下文")
	}
	file := r.GetUploadFile("file")
	if file == nil {
		return nil, xerror.New(xerror.CodeBusinessError, "请选择要上传的文件")
	}
	f, err := file.Open()
	if err != nil {
		return nil, xerror.Wrap(xerror.CodeBusinessError, err, "打开文件失败")
	}
	defer f.Close()

	return service.File().Upload(ctx, f, file.FileHeader)
}

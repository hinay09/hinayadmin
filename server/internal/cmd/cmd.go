package cmd

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	"hinay.cn/admin/internal/controller/system"

	"hinay.cn/admin/internal/controller/auth"
	"hinay.cn/admin/internal/controller/message"
	"hinay.cn/admin/internal/logic/casbinx"
	"hinay.cn/admin/internal/middleware"
)

// Main 启动 HTTP 服务。
var Main = gcmd.Command{
	Name:  "main",
	Usage: "main",
	Brief: "start http server",
	Func: func(ctx context.Context, _ *gcmd.Parser) error {
		// 预加载 Casbin 策略 (失败仅打日志, 不阻断启动)
		if _, err := casbinx.Reload(ctx); err != nil {
			g.Log().Warningf(ctx, "casbin reload on startup failed: %v", err)
		}

		s := g.Server()

		// 健康检查
		s.BindHandler("GET:/health", func(r *ghttp.Request) {
			r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok"})
		})

		// API 路由
		s.Group("/api/v1", func(grp *ghttp.RouterGroup) {
			// 注意: OperationLog 必须在 ghttp.MiddlewareHandlerResponse 之外层注册,
			// 其 defer 阶段才能读到最终响应的业务码与 HTTP 状态
			grp.Middleware(
				middleware.CORS,
				middleware.RequestId,
				middleware.SecurityHeaders,
				middleware.OperationLog,
				ghttp.MiddlewareHandlerResponse,
			)

			// 鉴权
			grp.Group("/", func(sec *ghttp.RouterGroup) {
				sec.Middleware(middleware.Auth, middleware.Casbin)
				sec.Bind(
					auth.NewV1(),
					message.NewV1(),
					system.NewV1(),
				)
			})
		})

		// 上传文件静态服务(替代 AddStaticPath, 以便附加安全响应头)。
		// 上传侧已做扩展名白名单与内容嗅探拦截(见 logic/system/file.go), 此处再加
		// nosniff + 非图片强制下载, 纵深防御浏览器将上传内容当页面执行(存储型 XSS)。
		s.BindHandler("/upload/*path", serveUploadFile)

		s.Run()
		return nil
	},
}

// uploadRoot 上传文件物理根目录。
const uploadRoot = "resource/upload"

// inlineUploadExts 允许浏览器直接内联展示的扩展名; 其余类型一律作为附件下载。
var inlineUploadExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".bmp": true, ".ico": true,
}

// serveUploadFile 带安全头的上传文件静态服务。
// 安全约束: 路径清洗后必须仍位于 uploadRoot 之下, 阻止 ../ 路径穿越。
func serveUploadFile(r *ghttp.Request) {
	root, err := filepath.Abs(uploadRoot)
	if err != nil {
		r.Response.WriteStatus(http.StatusInternalServerError)
		return
	}
	rel := r.Get("path").String()
	// 以 "/" 前缀做 Clean, 使 ".." 无法爬出根目录
	full := filepath.Join(root, filepath.Clean("/"+rel))
	if !strings.HasPrefix(full, root+string(filepath.Separator)) {
		r.Response.WriteStatus(http.StatusForbidden)
		return
	}
	fi, serr := os.Stat(full)
	if serr != nil || fi.IsDir() {
		r.Response.WriteStatus(http.StatusNotFound)
		return
	}

	h := r.Response.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cache-Control", "public, max-age=86400")
	if inlineUploadExts[strings.ToLower(filepath.Ext(full))] {
		h.Set("Content-Disposition", "inline")
	} else {
		h.Set("Content-Disposition", "attachment")
	}
	r.Response.ServeFile(full)
}

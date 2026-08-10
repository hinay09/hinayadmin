package cmd

import (
	"context"

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
			grp.Middleware(
				middleware.CORS,
				middleware.RequestId,
				ghttp.MiddlewareHandlerResponse,
			)

			// 鉴权
			grp.Group("/", func(sec *ghttp.RouterGroup) {
				sec.Middleware(middleware.Auth, middleware.Casbin, middleware.OperationLog)
				sec.Bind(
					auth.NewV1(),
					message.NewV1(),
					system.NewV1(),
				)
			})
		})

		// 文件上传静态路径
		// upload 文件存储于 resource/upload, 通过 /upload/ 路径直接访问(不经过鉴权中间件)
		s.AddStaticPath("/upload", "resource/upload")

		s.Run()
		return nil
	},
}

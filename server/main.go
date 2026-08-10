package main

import (
	_ "hinay.cn/admin/internal/packed"

	_ "github.com/gogf/gf/contrib/nosql/redis/v2"
	_ "hinay.cn/admin/internal/logic"

	"github.com/gogf/gf/v2/os/gctx"
	"hinay.cn/admin/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}

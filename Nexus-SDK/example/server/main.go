package main

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	nexus "github.com/krustd/nexus-sdk"
)

func main() {
	// ✅ 一行注册，配置全在 TOML 里
	nexus.MustSetup("config/config.toml")
	defer nexus.Shutdown()

	// 正常启动 gf server
	s := g.Server()
	s.BindHandler("/", func(r *ghttp.Request) {
		r.Response.Write("Hello from user-service")
	})
	s.BindHandler("/health", func(r *ghttp.Request) {
		r.Response.WriteJson(g.Map{"status": "ok"})
	})
	s.SetPort(8080)
	s.Run()
}

package main

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	nexus "github.com/krustd/gf-nexus/nexus-registry"
)

func main() {
	nexus.MustSetup("config/config.toml")
	defer nexus.Shutdown()

	s := g.Server()
	s.BindHandler("/", func(r *ghttp.Request) {
		r.Response.Write("Hello from user-service")
	})
	s.SetPort(8080)
	s.Run()
}

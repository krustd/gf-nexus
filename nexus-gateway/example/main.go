package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	nexus "github.com/krustd/gf-nexus/nexus-gateway"
)

func main() {
	configPath := "config.toml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	nexus.MustSetup(configPath)

	// 优雅关闭
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("[nexus-gateway] shutting down...")
		nexus.Shutdown()
		os.Exit(0)
	}()

	// 阻塞启动
	nexus.Start()
}

package registry

import (
	"context"
	"sync"
	"time"
)

var (
	globalRegistry Registry
	globalMu       sync.RWMutex
)

// SetGlobal 设置全局注册中心
func SetGlobal(r Registry) {
	globalMu.Lock()
	globalRegistry = r
	globalMu.Unlock()
}

// GetGlobal 获取全局注册中心
func GetGlobal() Registry {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalRegistry
}

// Shutdown 优雅关闭全局注册中心
func Shutdown() error {
	globalMu.Lock()
	reg := globalRegistry
	globalRegistry = nil
	globalMu.Unlock()
	if reg != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return reg.Close(ctx)
	}
	return nil
}

package registry

import (
	"context"
	"sync"
	"time"
)

var (
	globalRegistry Registry // ← 接口类型，不是 *Registry
	globalMu       sync.RWMutex
)

// SetGlobal 设置全局注册中心（传入接口值）
func SetGlobal(r Registry) {
	globalMu.Lock()
	globalRegistry = r
	globalMu.Unlock()
}

// GetGlobal 获取全局注册中心（返回接口值）
func GetGlobal() Registry {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalRegistry
}

// Shutdown 关闭全局注册中心
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

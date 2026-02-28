package config

import (
	"log"
	"sync"
	"sync/atomic"
)

// DynamicConfigHolder 动态配置持有器，使用 atomic.Pointer 保证并发安全
type DynamicConfigHolder struct {
	cfg       atomic.Pointer[DynamicConfig]
	callbacks []func(*DynamicConfig)
	mu        sync.RWMutex
}

// NewDynamicConfigHolder 创建持有器
func NewDynamicConfigHolder() *DynamicConfigHolder {
	h := &DynamicConfigHolder{}
	// 初始存入空默认配置
	def := &DynamicConfig{}
	ApplyDynamicDefaults(def)
	h.cfg.Store(def)
	return h
}

// Load 读取当前动态配置（无锁，适合每次请求调用）
func (h *DynamicConfigHolder) Load() *DynamicConfig {
	return h.cfg.Load()
}

// Store 原子替换动态配置，并触发变更回调
func (h *DynamicConfigHolder) Store(cfg *DynamicConfig) {
	ApplyDynamicDefaults(cfg)
	h.cfg.Store(cfg)

	h.mu.RLock()
	cbs := make([]func(*DynamicConfig), len(h.callbacks))
	copy(cbs, h.callbacks)
	h.mu.RUnlock()

	for _, cb := range cbs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[nexus-gateway] dynamic config callback panic: %v", r)
				}
			}()
			cb(cfg)
		}()
	}
}

// OnChange 注册配置变更回调（在 Store 时同步调用）
func (h *DynamicConfigHolder) OnChange(cb func(*DynamicConfig)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.callbacks = append(h.callbacks, cb)
}

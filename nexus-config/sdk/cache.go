package sdk

import (
	"sync"

	"github.com/krustd/gf-nexus/nexus-config/common"
)

// ConfigCache 配置本地缓存
type ConfigCache struct {
	mu      sync.RWMutex
	configs map[string]*common.ConfigVersion // key: namespace/key
}

func NewConfigCache() *ConfigCache {
	return &ConfigCache{
		configs: make(map[string]*common.ConfigVersion),
	}
}

// Get 获取缓存的配置
func (c *ConfigCache) Get(namespace, key string) (*common.ConfigVersion, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	configKey := namespace + "/" + key
	version, ok := c.configs[configKey]
	return version, ok
}

// Set 设置缓存
func (c *ConfigCache) Set(version *common.ConfigVersion) {
	c.mu.Lock()
	defer c.mu.Unlock()

	configKey := version.Namespace + "/" + version.Key
	c.configs[configKey] = version
}

// Delete 删除缓存
func (c *ConfigCache) Delete(namespace, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	configKey := namespace + "/" + key
	delete(c.configs, configKey)
}

// Clear 清空缓存
func (c *ConfigCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.configs = make(map[string]*common.ConfigVersion)
}

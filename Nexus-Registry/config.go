package registry

import (
	"fmt"
	"time"
)

// Config 注册中心配置
type Config struct {
	// etcd 集群地址
	Endpoints []string `json:"endpoints"`

	// 连接超时
	DialTimeout time.Duration `json:"dial_timeout"`

	// 服务租约 TTL（秒），实例多久没心跳就过期
	// 底层 etcd lease 的 TTL，KeepAlive 会按 TTL/3 自动续租
	LeaseTTL int64 `json:"lease_ttl"`

	// etcd key 前缀
	Prefix string `json:"prefix"`

	// etcd 认证（可选）
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
		LeaseTTL:    15, // 15 秒
		Prefix:      "/nexus/services",
	}
}

// Validate 校验配置
func (c *Config) Validate() error {
	if len(c.Endpoints) == 0 {
		return fmt.Errorf("nexus-registry: endpoints cannot be empty")
	}
	if c.DialTimeout <= 0 {
		c.DialTimeout = 5 * time.Second
	}
	if c.LeaseTTL <= 0 {
		c.LeaseTTL = 15
	}
	if c.Prefix == "" {
		c.Prefix = "/nexus/services"
	}
	return nil
}

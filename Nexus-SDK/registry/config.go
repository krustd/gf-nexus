package registry

import (
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

// Config 注册中心配置（对应 TOML 中的 [nexus.registry] 段）
type Config struct {
	// etcd 集群地址
	Endpoints []string `toml:"endpoints" json:"endpoints"`

	// 连接超时（秒）
	DialTimeoutSec int `toml:"dial_timeout" json:"dial_timeout"`

	// 租约 TTL（秒），实例多久没心跳就过期
	LeaseTTL int64 `toml:"lease_ttl" json:"lease_ttl"`

	// etcd key 前缀
	Prefix string `toml:"prefix" json:"prefix"`

	// etcd 认证（可选）
	Username string `toml:"username,omitempty" json:"username,omitempty"`
	Password string `toml:"password,omitempty" json:"password,omitempty"`
}

// ServiceConfig 当前服务自身的配置（对应 TOML 中的 [nexus.service] 段）
type ServiceConfig struct {
	Name     string            `toml:"name"     json:"name"`
	Version  string            `toml:"version"  json:"version"`
	Protocol string            `toml:"protocol" json:"protocol"` // http / grpc
	Address  string            `toml:"address"  json:"address"`  // 监听地址 host:port
	Weight   int               `toml:"weight"   json:"weight"`
	Metadata map[string]string `toml:"metadata" json:"metadata,omitempty"`
}

// NexusConfig 完整的 Nexus 配置块
type NexusConfig struct {
	Registry Config        `toml:"registry"`
	Service  ServiceConfig `toml:"service"`
}

// tomlRoot TOML 文件的根结构
type tomlRoot struct {
	Nexus NexusConfig `toml:"nexus"`
}

// DialTimeout 返回 time.Duration
func (c *Config) DialTimeout() time.Duration {
	if c.DialTimeoutSec <= 0 {
		return 5 * time.Second
	}
	return time.Duration(c.DialTimeoutSec) * time.Second
}

// Validate 校验配置
func (c *Config) Validate() error {
	if len(c.Endpoints) == 0 {
		return fmt.Errorf("nexus: endpoints cannot be empty")
	}
	if c.LeaseTTL <= 0 {
		c.LeaseTTL = 15
	}
	if c.Prefix == "" {
		c.Prefix = "/nexus/services"
	}
	return nil
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Endpoints:      []string{"127.0.0.1:2379"},
		DialTimeoutSec: 5,
		LeaseTTL:       15,
		Prefix:         "/nexus/services",
	}
}

// LoadConfig 从 TOML 文件加载配置
func LoadConfig(path string) (*NexusConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("nexus: read config file %s: %w", path, err)
	}

	var root tomlRoot
	if err := toml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("nexus: parse toml %s: %w", path, err)
	}

	// 填充默认值
	conf := &root.Nexus
	if len(conf.Registry.Endpoints) == 0 {
		conf.Registry.Endpoints = []string{"127.0.0.1:2379"}
	}
	if conf.Registry.DialTimeoutSec <= 0 {
		conf.Registry.DialTimeoutSec = 5
	}
	if conf.Registry.LeaseTTL <= 0 {
		conf.Registry.LeaseTTL = 15
	}
	if conf.Registry.Prefix == "" {
		conf.Registry.Prefix = "/nexus/services"
	}
	if conf.Service.Weight <= 0 {
		conf.Service.Weight = 1
	}
	if conf.Service.Protocol == "" {
		conf.Service.Protocol = "http"
	}

	return conf, nil
}

// ToInstance 将 ServiceConfig 转为 ServiceInstance
func (sc *ServiceConfig) ToInstance() *ServiceInstance {
	inst := &ServiceInstance{
		Name:     sc.Name,
		Version:  sc.Version,
		Protocol: Protocol(sc.Protocol),
		Address:  sc.Address,
		Weight:   sc.Weight,
		Metadata: sc.Metadata,
	}
	// 默认 ID = name-address
	inst.ID = fmt.Sprintf("%s-%s", sc.Name, sc.Address)
	return inst
}

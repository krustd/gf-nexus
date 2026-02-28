package registry

import (
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Endpoints      []string `toml:"endpoints"      json:"endpoints"`
	DialTimeoutSec int      `toml:"dial_timeout"   json:"dial_timeout"`
	LeaseTTL       int64    `toml:"lease_ttl"      json:"lease_ttl"`
	Prefix         string   `toml:"prefix"         json:"prefix"`
	Username       string   `toml:"username,omitempty" json:"username,omitempty"`
	Password       string   `toml:"password,omitempty" json:"password,omitempty"`
}

type ServiceConfig struct {
	Name     string            `toml:"name"     json:"name"`
	Version  string            `toml:"version"  json:"version"`
	Protocol string            `toml:"protocol" json:"protocol"`
	Address  string            `toml:"address"  json:"address"`
	Weight   int               `toml:"weight"   json:"weight"`
	Metadata map[string]string `toml:"metadata" json:"metadata,omitempty"`
}

type NexusConfig struct {
	Registry Config        `toml:"registry"`
	Service  ServiceConfig `toml:"service"`
}

type tomlRoot struct {
	Nexus NexusConfig `toml:"nexus"`
}

func (c *Config) DialTimeout() time.Duration {
	if c.DialTimeoutSec <= 0 {
		return 5 * time.Second
	}
	return time.Duration(c.DialTimeoutSec) * time.Second
}

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

func DefaultConfig() *Config {
	return &Config{
		Endpoints:      []string{"127.0.0.1:2379"},
		DialTimeoutSec: 5,
		LeaseTTL:       15,
		Prefix:         "/nexus/services",
	}
}

func LoadConfig(path string) (*NexusConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("nexus: read config %s: %w", path, err)
	}
	var root tomlRoot
	if err := toml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("nexus: parse toml %s: %w", path, err)
	}
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

func (sc *ServiceConfig) ToInstance() *ServiceInstance {
	inst := &ServiceInstance{
		Name:     sc.Name,
		Version:  sc.Version,
		Protocol: Protocol(sc.Protocol),
		Address:  sc.Address,
		Weight:   sc.Weight,
		Metadata: sc.Metadata,
	}
	inst.ID = fmt.Sprintf("%s-%s", sc.Name, sc.Address)
	return inst
}

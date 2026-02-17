package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// GatewayConfig 本地 TOML 静态配置（启动时确定，极少变更）
type GatewayConfig struct {
	Registry     RegistryConfig     `toml:"registry"`
	Server       ServerConfig       `toml:"server"`
	ConfigCenter ConfigCenterConfig `toml:"config_center"`
	Timeout      TimeoutConfig      `toml:"timeout"`
	Metrics      MetricsConfig      `toml:"metrics"`
	GRPC         GRPCConfig         `toml:"grpc"`
}

// RegistryConfig etcd 注册中心连接（配置中心的前置依赖）
type RegistryConfig struct {
	Endpoints      []string `toml:"endpoints"`
	DialTimeoutSec int      `toml:"dial_timeout"`
	Prefix         string   `toml:"prefix"`
	Username       string   `toml:"username,omitempty"`
	Password       string   `toml:"password,omitempty"`
}

// ServerConfig 服务监听地址
type ServerConfig struct {
	Addr string `toml:"addr"`
}

// ConfigCenterConfig 配置中心 SDK 连接参数
type ConfigCenterConfig struct {
	ServerAddr  string `toml:"server_addr"`
	Namespace   string `toml:"namespace"`
	ConfigKey   string `toml:"config_key"`
	ClientID    string `toml:"client_id"`
	PollTimeout int    `toml:"poll_timeout"`
	RetryDelay  int    `toml:"retry_delay"`
}

type TimeoutConfig struct {
	ConnectMs  int `toml:"connect_ms"`
	ResponseMs int `toml:"response_ms"`
}

type MetricsConfig struct {
	Enabled bool   `toml:"enabled"`
	Path    string `toml:"path"`
}

type GRPCConfig struct {
	ReflectionCacheTTLSec int `toml:"reflection_cache_ttl_sec"`
	ConnectTimeoutMs      int `toml:"connect_timeout_ms"`
	RequestTimeoutMs      int `toml:"request_timeout_ms"`
}

// ─── 动态配置（从配置中心获取，YAML 格式，支持热更新）───

// DynamicConfig 放在配置中心的运行时配置
type DynamicConfig struct {
	JWT       JWTConfig       `yaml:"jwt"       json:"jwt"`
	IPFilter  IPFilterConfig  `yaml:"ip_filter"  json:"ip_filter"`
	RateLimit RateLimitConfig `yaml:"rate_limit" json:"rate_limit"`
	Circuit   CircuitConfig   `yaml:"circuit"    json:"circuit"`
	CORS      CORSConfig      `yaml:"cors"       json:"cors"`
	Balancer  BalancerConfig  `yaml:"balancer"   json:"balancer"`
}

// JWTConfig 支持非对称加密 + JWKS 多密钥轮换
type JWTConfig struct {
	Enabled   bool      `yaml:"enabled"    json:"enabled"`
	Keys      []JWKItem `yaml:"keys"       json:"keys"`
	SkipPaths []string  `yaml:"skip_paths" json:"skip_paths"`
}

// JWKItem 单个公钥条目
type JWKItem struct {
	KID       string `yaml:"kid"        json:"kid"`        // Key ID，JWT header 中的 kid 字段
	Algorithm string `yaml:"algorithm"  json:"algorithm"`  // RS256 / EdDSA
	PublicKey string `yaml:"public_key" json:"public_key"` // PEM 编码的公钥
}

type IPFilterConfig struct {
	Enabled   bool     `yaml:"enabled"   json:"enabled"`
	Mode      string   `yaml:"mode"      json:"mode"` // whitelist / blacklist
	Addresses []string `yaml:"addresses" json:"addresses"`
}

type RateLimitConfig struct {
	Enabled bool    `yaml:"enabled" json:"enabled"`
	Rate    float64 `yaml:"rate"    json:"rate"`  // tokens per second
	Burst   int     `yaml:"burst"   json:"burst"` // bucket capacity
}

type CircuitConfig struct {
	Enabled        bool    `yaml:"enabled"         json:"enabled"`
	ErrorThreshold float64 `yaml:"error_threshold" json:"error_threshold"` // 0.0 ~ 1.0
	MinRequests    int     `yaml:"min_requests"    json:"min_requests"`
	WindowSec      int     `yaml:"window_sec"      json:"window_sec"`
	CooldownSec    int     `yaml:"cooldown_sec"    json:"cooldown_sec"`
}

type CORSConfig struct {
	Enabled          bool     `yaml:"enabled"           json:"enabled"`
	AllowOrigins     []string `yaml:"allow_origins"     json:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods"     json:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers"     json:"allow_headers"`
	AllowCredentials bool     `yaml:"allow_credentials" json:"allow_credentials"`
	MaxAgeSec        int      `yaml:"max_age_sec"       json:"max_age_sec"`
}

type BalancerConfig struct {
	Strategy string `yaml:"strategy" json:"strategy"` // round_robin / random / weighted_round_robin
}

// ─── 加载 & 默认值 ───

type tomlRoot struct {
	Gateway GatewayConfig `toml:"gateway"`
}

func LoadConfig(path string) (*GatewayConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("nexus-gateway: read config %s: %w", path, err)
	}
	var root tomlRoot
	if err := toml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("nexus-gateway: parse toml %s: %w", path, err)
	}
	cfg := &root.Gateway
	applyDefaults(cfg)
	return cfg, nil
}

func applyDefaults(cfg *GatewayConfig) {
	// Registry
	if len(cfg.Registry.Endpoints) == 0 {
		cfg.Registry.Endpoints = []string{"127.0.0.1:2379"}
	}
	if cfg.Registry.DialTimeoutSec <= 0 {
		cfg.Registry.DialTimeoutSec = 5
	}
	if cfg.Registry.Prefix == "" {
		cfg.Registry.Prefix = "/nexus/services"
	}

	// Server
	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}

	// ConfigCenter
	if cfg.ConfigCenter.Namespace == "" {
		cfg.ConfigCenter.Namespace = "nexus-gateway"
	}
	if cfg.ConfigCenter.ConfigKey == "" {
		cfg.ConfigCenter.ConfigKey = "gateway.yaml"
	}
	if cfg.ConfigCenter.PollTimeout <= 0 {
		cfg.ConfigCenter.PollTimeout = 30
	}
	if cfg.ConfigCenter.RetryDelay <= 0 {
		cfg.ConfigCenter.RetryDelay = 5
	}

	// Timeout
	if cfg.Timeout.ConnectMs <= 0 {
		cfg.Timeout.ConnectMs = 3000
	}
	if cfg.Timeout.ResponseMs <= 0 {
		cfg.Timeout.ResponseMs = 10000
	}

	// Metrics
	if cfg.Metrics.Path == "" {
		cfg.Metrics.Path = "/metrics"
	}

	// GRPC
	if cfg.GRPC.ReflectionCacheTTLSec <= 0 {
		cfg.GRPC.ReflectionCacheTTLSec = 300
	}
	if cfg.GRPC.ConnectTimeoutMs <= 0 {
		cfg.GRPC.ConnectTimeoutMs = 3000
	}
	if cfg.GRPC.RequestTimeoutMs <= 0 {
		cfg.GRPC.RequestTimeoutMs = 10000
	}
}

// ApplyDynamicDefaults 为动态配置填充默认值
func ApplyDynamicDefaults(cfg *DynamicConfig) {
	// RateLimit
	if cfg.RateLimit.Rate <= 0 {
		cfg.RateLimit.Rate = 1000
	}
	if cfg.RateLimit.Burst <= 0 {
		cfg.RateLimit.Burst = 2000
	}

	// Circuit
	if cfg.Circuit.ErrorThreshold <= 0 {
		cfg.Circuit.ErrorThreshold = 0.5
	}
	if cfg.Circuit.MinRequests <= 0 {
		cfg.Circuit.MinRequests = 20
	}
	if cfg.Circuit.WindowSec <= 0 {
		cfg.Circuit.WindowSec = 30
	}
	if cfg.Circuit.CooldownSec <= 0 {
		cfg.Circuit.CooldownSec = 15
	}

	// CORS
	if len(cfg.CORS.AllowOrigins) == 0 {
		cfg.CORS.AllowOrigins = []string{"*"}
	}
	if len(cfg.CORS.AllowMethods) == 0 {
		cfg.CORS.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	}
	if len(cfg.CORS.AllowHeaders) == 0 {
		cfg.CORS.AllowHeaders = []string{"Content-Type", "Authorization", "X-Request-Id"}
	}
	if cfg.CORS.MaxAgeSec <= 0 {
		cfg.CORS.MaxAgeSec = 3600
	}

	// Balancer
	if cfg.Balancer.Strategy == "" {
		cfg.Balancer.Strategy = "round_robin"
	}
}

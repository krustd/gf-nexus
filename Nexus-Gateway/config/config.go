package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type GatewayConfig struct {
	Registry  RegistryConfig  `toml:"registry"`
	Server    ServerConfig    `toml:"server"`
	Balancer  BalancerConfig  `toml:"balancer"`
	JWT       JWTConfig       `toml:"jwt"`
	IPFilter  IPFilterConfig  `toml:"ip_filter"`
	RateLimit RateLimitConfig `toml:"rate_limit"`
	Circuit   CircuitConfig   `toml:"circuit"`
	Timeout   TimeoutConfig   `toml:"timeout"`
	CORS      CORSConfig      `toml:"cors"`
	Metrics   MetricsConfig   `toml:"metrics"`
}

type RegistryConfig struct {
	Endpoints      []string `toml:"endpoints"`
	DialTimeoutSec int      `toml:"dial_timeout"`
	Prefix         string   `toml:"prefix"`
	Username       string   `toml:"username,omitempty"`
	Password       string   `toml:"password,omitempty"`
}

type ServerConfig struct {
	Addr string `toml:"addr"`
}

type BalancerConfig struct {
	Strategy string `toml:"strategy"` // round_robin / random / weighted_round_robin
}

type JWTConfig struct {
	Enabled   bool     `toml:"enabled"`
	Secret    string   `toml:"secret"`
	PublicKey string   `toml:"public_key"`
	Algorithm string   `toml:"algorithm"` // HS256 / RS256
	SkipPaths []string `toml:"skip_paths"`
}

type IPFilterConfig struct {
	Enabled   bool     `toml:"enabled"`
	Mode      string   `toml:"mode"` // whitelist / blacklist
	Addresses []string `toml:"addresses"`
}

type RateLimitConfig struct {
	Enabled bool    `toml:"enabled"`
	Rate    float64 `toml:"rate"`  // tokens per second
	Burst   int     `toml:"burst"` // bucket capacity
}

type CircuitConfig struct {
	Enabled        bool    `toml:"enabled"`
	ErrorThreshold float64 `toml:"error_threshold"` // 0.0 ~ 1.0
	MinRequests    int     `toml:"min_requests"`
	WindowSec      int     `toml:"window_sec"`
	CooldownSec    int     `toml:"cooldown_sec"`
}

type TimeoutConfig struct {
	ConnectMs  int `toml:"connect_ms"`
	ResponseMs int `toml:"response_ms"`
}

type CORSConfig struct {
	Enabled          bool     `toml:"enabled"`
	AllowOrigins     []string `toml:"allow_origins"`
	AllowMethods     []string `toml:"allow_methods"`
	AllowHeaders     []string `toml:"allow_headers"`
	AllowCredentials bool     `toml:"allow_credentials"`
	MaxAgeSec        int      `toml:"max_age_sec"`
}

type MetricsConfig struct {
	Enabled bool   `toml:"enabled"`
	Path    string `toml:"path"`
}

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

	// Balancer
	if cfg.Balancer.Strategy == "" {
		cfg.Balancer.Strategy = "round_robin"
	}

	// JWT
	if cfg.JWT.Algorithm == "" {
		cfg.JWT.Algorithm = "HS256"
	}

	// Rate Limit
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

	// Timeout
	if cfg.Timeout.ConnectMs <= 0 {
		cfg.Timeout.ConnectMs = 3000
	}
	if cfg.Timeout.ResponseMs <= 0 {
		cfg.Timeout.ResponseMs = 10000
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

	// Metrics
	if cfg.Metrics.Path == "" {
		cfg.Metrics.Path = "/metrics"
	}
}

package common

import (
	"fmt"
	"os"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gfile"
)

// ServerConfig 配置中心服务端配置
type ServerConfig struct {
	Database DatabaseConfig `json:"database"`
	Admin    AdminConfig    `json:"admin"`
	Server   HttpConfig     `json:"server"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type     string `json:"type"`     // sqlite, mysql
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	FilePath string `json:"file_path"` // for sqlite
}

// AdminConfig Admin API 配置
type AdminConfig struct {
	Addr string `json:"addr"`
}

// HttpConfig HTTP 服务配置
type HttpConfig struct {
	Addr string `json:"addr"`
}

// ClientConfig SDK 客户端配置
type ClientConfig struct {
	ServerAddr  string `json:"server_addr"`
	Namespace   string `json:"namespace"`
	ConfigKey   string `json:"config_key"`
	ClientID    string `json:"client_id"`     // 客户端唯一标识，用于灰度
	PollTimeout int    `json:"poll_timeout"`  // 长轮询超时时间（秒）
	RetryDelay  int    `json:"retry_delay"`   // 重试延迟（秒）
}

// LoadServerConfig 加载服务端配置
func LoadServerConfig(path string) (*ServerConfig, error) {
	if !gfile.Exists(path) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	content := gfile.GetContents(path)
	j, err := gjson.LoadContent([]byte(content))
	if err != nil {
		return nil, fmt.Errorf("parse config failed: %w", err)
	}

	var cfg ServerConfig
	if err := j.Scan(&cfg); err != nil {
		return nil, fmt.Errorf("scan config failed: %w", err)
	}

	return &cfg, nil
}

// LoadClientConfig 加载客户端配置
func LoadClientConfig(path string) (*ClientConfig, error) {
	if !gfile.Exists(path) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	content := gfile.GetContents(path)
	j, err := gjson.LoadContent([]byte(content))
	if err != nil {
		return nil, fmt.Errorf("parse config failed: %w", err)
	}

	var cfg ClientConfig
	if err := j.Scan(&cfg); err != nil {
		return nil, fmt.Errorf("scan config failed: %w", err)
	}

	// 自动获取客户端 ID（如果未配置）
	if cfg.ClientID == "" {
		hostname, _ := os.Hostname()
		cfg.ClientID = hostname
	}

	// 设置默认值
	if cfg.PollTimeout == 0 {
		cfg.PollTimeout = 30
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 5
	}

	return &cfg, nil
}

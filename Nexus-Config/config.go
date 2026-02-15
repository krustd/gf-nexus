package main

import (
	"context"

	"github.com/krustd/nexus-config/common"
	"github.com/krustd/nexus-config/sdk"
)

var (
	globalClient *sdk.Client
)

// Setup 初始化配置中心客户端
func Setup(configPath string) error {
	cfg, err := common.LoadClientConfig(configPath)
	if err != nil {
		return err
	}

	globalClient = sdk.NewClient(cfg)
	return globalClient.Start(context.Background())
}

// MustSetup 初始化配置中心客户端（失败则 panic）
func MustSetup(configPath string) {
	if err := Setup(configPath); err != nil {
		panic(err)
	}
}

// Shutdown 关闭配置中心客户端
func Shutdown() {
	if globalClient != nil {
		globalClient.Stop()
	}
}

// GetClient 获取全局客户端实例
func GetClient() *sdk.Client {
	return globalClient
}

// GetConfig 获取当前配置
func GetConfig() (*common.ConfigVersion, error) {
	if globalClient == nil {
		return nil, ErrClientNotInitialized
	}
	return globalClient.GetConfig()
}

// GetValue 获取配置内容
func GetValue() (string, error) {
	if globalClient == nil {
		return "", ErrClientNotInitialized
	}
	return globalClient.GetValue()
}

// GetValueAs 获取配置并解析到目标对象
func GetValueAs(target interface{}) error {
	if globalClient == nil {
		return ErrClientNotInitialized
	}
	return globalClient.GetValueAs(target)
}

// AddChangeListener 添加配置变更监听器
func AddChangeListener(listener sdk.ChangeListener) {
	if globalClient != nil {
		globalClient.AddChangeListener(listener)
	}
}

var (
	ErrClientNotInitialized = &Error{Code: 1001, Message: "config client not initialized"}
)

type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

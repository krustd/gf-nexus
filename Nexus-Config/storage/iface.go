package storage

import (
	"context"

	"github.com/krustd/gf-nexus/nexus-config/common"
)

// Storage 配置存储接口
type Storage interface {
	// Init 初始化存储（创建表等）
	Init(ctx context.Context) error

	// Close 关闭存储
	Close() error

	// === Namespace 操作 ===

	// CreateNamespace 创建命名空间
	CreateNamespace(ctx context.Context, ns *common.ConfigNamespace) error

	// GetNamespace 获取命名空间
	GetNamespace(ctx context.Context, id string) (*common.ConfigNamespace, error)

	// ListNamespaces 列出所有命名空间
	ListNamespaces(ctx context.Context) ([]*common.ConfigNamespace, error)

	// DeleteNamespace 删除命名空间
	DeleteNamespace(ctx context.Context, id string) error

	// === ConfigItem 操作 ===

	// SaveDraft 保存草稿
	SaveDraft(ctx context.Context, namespace, key string, value string, format common.ConfigFormat) error

	// GetDraft 获取草稿
	GetDraft(ctx context.Context, namespace, key string) (*common.ConfigItem, error)

	// PublishConfig 发布配置（将草稿发布为正式版本）
	PublishConfig(ctx context.Context, namespace, key string) error

	// GetPublishedConfig 获取已发布的配置
	GetPublishedConfig(ctx context.Context, namespace, key string) (*common.ConfigItem, error)

	// ListConfigs 列出命名空间下的所有配置
	ListConfigs(ctx context.Context, namespace string) ([]*common.ConfigItem, error)

	// DeleteConfig 删除配置项
	DeleteConfig(ctx context.Context, namespace, key string) error

	// === GrayRule 操作 ===

	// SaveGrayRule 保存灰度规则
	SaveGrayRule(ctx context.Context, rule *common.GrayRule) error

	// GetGrayRule 获取灰度规则
	GetGrayRule(ctx context.Context, namespace, key string) (*common.GrayRule, error)

	// DeleteGrayRule 删除灰度规则
	DeleteGrayRule(ctx context.Context, namespace, key string) error

	// ListGrayRules 列出命名空间下的所有灰度规则
	ListGrayRules(ctx context.Context, namespace string) ([]*common.GrayRule, error)
}

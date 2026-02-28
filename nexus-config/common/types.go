package common

import "time"

// ConfigFormat 配置格式
type ConfigFormat string

const (
	FormatYAML       ConfigFormat = "yaml"
	FormatJSON       ConfigFormat = "json"
	FormatTOML       ConfigFormat = "toml"
	FormatProperties ConfigFormat = "properties"
)

// ConfigNamespace 命名空间（对应一个应用）
type ConfigNamespace struct {
	ID          string    `json:"id" gorm:"primaryKey;size:64"`
	Name        string    `json:"name" gorm:"size:128;not null"`
	Description string    `json:"description" gorm:"size:512"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (ConfigNamespace) TableName() string {
	return "config_namespace"
}

// ConfigItem 配置项
type ConfigItem struct {
	ID             int64        `json:"id" gorm:"primaryKey;autoIncrement"`
	Namespace      string       `json:"namespace" gorm:"size:64;not null;index:idx_config_ns_key,unique"`
	Key            string       `json:"key" gorm:"size:128;not null;index:idx_config_ns_key,unique"`
	Format         ConfigFormat `json:"format" gorm:"size:20;default:yaml"`
	DraftValue     string       `json:"draft_value" gorm:"type:text"`
	DraftMD5       string       `json:"draft_md5" gorm:"size:32"`
	PublishedValue string       `json:"published_value" gorm:"type:text"`
	PublishedMD5   string       `json:"published_md5" gorm:"size:32"`
	PublishedAt    *time.Time   `json:"published_at"`
	CreatedAt      time.Time    `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time    `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (ConfigItem) TableName() string {
	return "config_item"
}

// GrayRule 灰度规则
type GrayRule struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Namespace  string    `json:"namespace" gorm:"size:64;not null;index:idx_gray_ns_key,unique"`
	Key        string    `json:"key" gorm:"size:128;not null;index:idx_gray_ns_key,unique"`
	Percentage int       `json:"percentage" gorm:"default:0"` // 0-100
	Enabled    bool      `json:"enabled" gorm:"default:false"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (GrayRule) TableName() string {
	return "gray_rule"
}

// ConfigVersion 配置版本（用于长轮询）
type ConfigVersion struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	MD5       string `json:"md5"`
	Value     string `json:"value"`
	Format    string `json:"format"`
}

// WatchEvent 配置变更事件
type WatchEvent struct {
	Namespace string       `json:"namespace"`
	Key       string       `json:"key"`
	EventType WatchEventType `json:"event_type"`
	Version   *ConfigVersion `json:"version,omitempty"`
}

// WatchEventType 事件类型
type WatchEventType string

const (
	EventTypeUpdate WatchEventType = "update"
	EventTypeDelete WatchEventType = "delete"
)

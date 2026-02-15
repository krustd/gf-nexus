package registry

import (
	"encoding/json"
	"fmt"
)

// Protocol 服务协议类型
type Protocol string

const (
	ProtocolHTTP Protocol = "http"
	ProtocolGRPC Protocol = "grpc"
)

// EventType Watch 事件类型
type EventType int

const (
	EventTypePut    EventType = iota // 新增或更新
	EventTypeDelete                  // 删除（下线）
)

// WatchEvent 服务变更事件
type WatchEvent struct {
	Type     EventType
	Instance *ServiceInstance
}

// ServiceInstance 代表一个服务实例
type ServiceInstance struct {
	ID       string            `json:"id"`                 // 实例唯一 ID
	Name     string            `json:"name"`               // 服务名称
	Version  string            `json:"version"`            // 服务版本
	Protocol Protocol          `json:"protocol"`           // http / grpc
	Address  string            `json:"address"`            // 监听地址 host:port
	Weight   int               `json:"weight"`             // 权重，默认 1
	Metadata map[string]string `json:"metadata,omitempty"` // 扩展元数据
}

// Validate 校验实例字段
func (s *ServiceInstance) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("nexus: service name cannot be empty")
	}
	if s.Address == "" {
		return fmt.Errorf("nexus: service address cannot be empty")
	}
	if s.ID == "" {
		s.ID = s.Address
	}
	if s.Protocol == "" {
		s.Protocol = ProtocolHTTP
	}
	if s.Weight <= 0 {
		s.Weight = 1
	}
	return nil
}

// Marshal 序列化为 JSON
func (s *ServiceInstance) Marshal() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("nexus: marshal instance: %w", err)
	}
	return string(data), nil
}

// UnmarshalInstance 反序列化
func UnmarshalInstance(data []byte) (*ServiceInstance, error) {
	var s ServiceInstance
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("nexus: unmarshal instance: %w", err)
	}
	return &s, nil
}

// BuildKey 构建 etcd key: {prefix}/{name}/{id}
func (s *ServiceInstance) BuildKey(prefix string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, s.Name, s.ID)
}

// ServicePrefix 服务前缀: {prefix}/{name}/
func ServicePrefix(prefix, name string) string {
	return fmt.Sprintf("%s/%s/", prefix, name)
}

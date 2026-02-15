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

// ServiceInstance 代表一个服务实例（注册到 etcd 的最小单元）
type ServiceInstance struct {
	// 基本信息
	ID        string   `json:"id"`        // 实例唯一 ID（一般用 host:port 或 uuid）
	Name      string   `json:"name"`      // 服务名称，如 "user-service"
	Version   string   `json:"version"`   // 服务版本，如 "v1.0.0"
	Protocol  Protocol `json:"protocol"`  // 协议类型：http / grpc
	Address   string   `json:"address"`   // 监听地址，如 "10.0.0.1:8080"
	Weight    int      `json:"weight"`    // 权重（用于加权负载均衡），默认 1
	Metadata  map[string]string `json:"metadata,omitempty"` // 扩展元数据
}

// Validate 校验实例字段
func (s *ServiceInstance) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("nexus-registry: service name cannot be empty")
	}
	if s.Address == "" {
		return fmt.Errorf("nexus-registry: service address cannot be empty")
	}
	if s.ID == "" {
		s.ID = s.Address // 默认用地址做 ID
	}
	if s.Protocol == "" {
		s.Protocol = ProtocolHTTP
	}
	if s.Weight <= 0 {
		s.Weight = 1
	}
	return nil
}

// Marshal 序列化为 JSON（存入 etcd value）
func (s *ServiceInstance) Marshal() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("nexus-registry: marshal service instance: %w", err)
	}
	return string(data), nil
}

// UnmarshalInstance 从 JSON 反序列化
func UnmarshalInstance(data []byte) (*ServiceInstance, error) {
	var s ServiceInstance
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("nexus-registry: unmarshal service instance: %w", err)
	}
	return &s, nil
}

// BuildKey 构建 etcd key
// 格式: /nexus/services/{name}/{id}
func (s *ServiceInstance) BuildKey(prefix string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, s.Name, s.ID)
}

// ServicePrefix 根据服务名获取 etcd 前缀（用于 Watch / 发现）
func ServicePrefix(prefix, name string) string {
	return fmt.Sprintf("%s/%s/", prefix, name)
}

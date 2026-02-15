package registry

import (
	"encoding/json"
	"fmt"
)

type Protocol string

const (
	ProtocolHTTP Protocol = "http"
	ProtocolGRPC Protocol = "grpc"
)

type EventType int

const (
	EventTypePut    EventType = iota
	EventTypeDelete
)

type WatchEvent struct {
	Type     EventType
	Instance *ServiceInstance
}

type ServiceInstance struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Protocol Protocol          `json:"protocol"`
	Address  string            `json:"address"`
	Weight   int               `json:"weight"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

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

func (s *ServiceInstance) Marshal() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("nexus: marshal: %w", err)
	}
	return string(data), nil
}

func UnmarshalInstance(data []byte) (*ServiceInstance, error) {
	var s ServiceInstance
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("nexus: unmarshal: %w", err)
	}
	return &s, nil
}

func (s *ServiceInstance) BuildKey(prefix string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, s.Name, s.ID)
}

func ServicePrefix(prefix, name string) string {
	return fmt.Sprintf("%s/%s/", prefix, name)
}

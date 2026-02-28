package server

import (
	"context"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/krustd/gf-nexus/nexus-config/common"
)

// ConfigNotifier 配置变更通知器
type ConfigNotifier struct {
	mu        sync.RWMutex
	listeners map[string][]chan *common.ConfigVersion // key: namespace/key
}

func NewConfigNotifier() *ConfigNotifier {
	return &ConfigNotifier{
		listeners: make(map[string][]chan *common.ConfigVersion),
	}
}

// Subscribe 订阅配置变更
func (n *ConfigNotifier) Subscribe(namespace, key string) <-chan *common.ConfigVersion {
	n.mu.Lock()
	defer n.mu.Unlock()

	configKey := namespace + "/" + key
	ch := make(chan *common.ConfigVersion, 1)
	n.listeners[configKey] = append(n.listeners[configKey], ch)

	return ch
}

// Unsubscribe 取消订阅
func (n *ConfigNotifier) Unsubscribe(namespace, key string, ch <-chan *common.ConfigVersion) {
	n.mu.Lock()
	defer n.mu.Unlock()

	configKey := namespace + "/" + key
	listeners := n.listeners[configKey]
	for i, listener := range listeners {
		if listener == ch {
			n.listeners[configKey] = append(listeners[:i], listeners[i+1:]...)
			close(listener)
			break
		}
	}

	if len(n.listeners[configKey]) == 0 {
		delete(n.listeners, configKey)
	}
}

// Notify 通知配置变更
func (n *ConfigNotifier) Notify(ctx context.Context, version *common.ConfigVersion) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	configKey := version.Namespace + "/" + version.Key
	listeners := n.listeners[configKey]

	g.Log().Infof(ctx, "notifying %d listeners for config: %s", len(listeners), configKey)

	for _, ch := range listeners {
		select {
		case ch <- version:
			// 成功发送
		default:
			// channel 满了，跳过
			g.Log().Warningf(ctx, "listener channel full for config: %s", configKey)
		}
	}
}

// WaitForChange 等待配置变更（带超时）
func (n *ConfigNotifier) WaitForChange(ctx context.Context, namespace, key string, timeout time.Duration) (*common.ConfigVersion, bool) {
	ch := n.Subscribe(namespace, key)
	defer n.Unsubscribe(namespace, key, ch)

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case version := <-ch:
		return version, true
	case <-timer.C:
		return nil, false
	case <-ctx.Done():
		return nil, false
	}
}

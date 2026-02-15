package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/gclient"
	"github.com/krustd/nexus-config/common"
)

// ChangeListener 配置变更监听器
type ChangeListener func(version *common.ConfigVersion)

// Client 配置中心客户端
type Client struct {
	cfg      *common.ClientConfig
	cache    *ConfigCache
	client   *gclient.Client
	listeners map[string][]ChangeListener // key: namespace/key
	mu       sync.RWMutex
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewClient 创建配置中心客户端
func NewClient(cfg *common.ClientConfig) *Client {
	return &Client{
		cfg:       cfg,
		cache:     NewConfigCache(),
		client:    g.Client(),
		listeners: make(map[string][]ChangeListener),
		stopCh:    make(chan struct{}),
	}
}

// Start 启动客户端（开始长轮询）
func (c *Client) Start(ctx context.Context) error {
	g.Log().Infof(ctx, "config client starting, namespace=%s, key=%s", c.cfg.Namespace, c.cfg.ConfigKey)

	// 首次拉取配置
	if err := c.fetchConfig(ctx); err != nil {
		g.Log().Warningf(ctx, "initial fetch config failed: %v", err)
	}

	// 启动长轮询
	c.wg.Add(1)
	go c.longPollLoop(ctx)

	return nil
}

// Stop 停止客户端
func (c *Client) Stop() {
	close(c.stopCh)
	c.wg.Wait()
	g.Log().Info(context.Background(), "config client stopped")
}

// GetConfig 获取配置（从本地缓存）
func (c *Client) GetConfig() (*common.ConfigVersion, error) {
	version, ok := c.cache.Get(c.cfg.Namespace, c.cfg.ConfigKey)
	if !ok {
		return nil, fmt.Errorf("config not found in cache")
	}
	return version, nil
}

// GetValue 获取配置内容
func (c *Client) GetValue() (string, error) {
	version, err := c.GetConfig()
	if err != nil {
		return "", err
	}
	return version.Value, nil
}

// GetValueAs 获取配置并解析到目标对象
func (c *Client) GetValueAs(target interface{}) error {
	version, err := c.GetConfig()
	if err != nil {
		return err
	}

	format := common.ConfigFormat(version.Format)
	return common.ParseConfig(version.Value, format, target)
}

// AddChangeListener 添加配置变更监听器
func (c *Client) AddChangeListener(listener ChangeListener) {
	c.mu.Lock()
	defer c.mu.Unlock()

	configKey := c.cfg.Namespace + "/" + c.cfg.ConfigKey
	c.listeners[configKey] = append(c.listeners[configKey], listener)
}

// longPollLoop 长轮询循环
func (c *Client) longPollLoop(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-c.stopCh:
			return
		default:
			c.pollOnce(ctx)
		}
	}
}

// pollOnce 执行一次长轮询
func (c *Client) pollOnce(ctx context.Context) {
	// 获取当前配置的 MD5
	currentMD5 := ""
	if version, ok := c.cache.Get(c.cfg.Namespace, c.cfg.ConfigKey); ok {
		currentMD5 = version.MD5
	}

	// 构造请求
	req := map[string]interface{}{
		"namespace": c.cfg.Namespace,
		"key":       c.cfg.ConfigKey,
		"client_id": c.cfg.ClientID,
		"md5":       currentMD5,
	}

	url := fmt.Sprintf("%s/api/v1/config/poll", c.cfg.ServerAddr)

	// 发起长轮询请求（超时时间设置为 poll_timeout + 5s）
	timeout := time.Duration(c.cfg.PollTimeout+5) * time.Second
	resp, err := c.client.Timeout(timeout).Post(ctx, url, req)
	if err != nil {
		g.Log().Errorf(ctx, "poll config failed: %v", err)
		time.Sleep(time.Duration(c.cfg.RetryDelay) * time.Second)
		return
	}
	defer resp.Close()

	// 解析响应
	var pollResp struct {
		Changed bool                   `json:"changed"`
		Version *common.ConfigVersion `json:"version"`
	}

	if err := json.Unmarshal(resp.ReadAll(), &pollResp); err != nil {
		g.Log().Errorf(ctx, "parse poll response failed: %v", err)
		time.Sleep(time.Duration(c.cfg.RetryDelay) * time.Second)
		return
	}

	// 如果配置变更
	if pollResp.Changed && pollResp.Version != nil {
		g.Log().Infof(ctx, "config changed: namespace=%s, key=%s, md5=%s",
			pollResp.Version.Namespace, pollResp.Version.Key, pollResp.Version.MD5)

		// 更新缓存
		c.cache.Set(pollResp.Version)

		// 通知监听器
		c.notifyListeners(pollResp.Version)
	}
}

// fetchConfig 立即拉取配置（非长轮询）
func (c *Client) fetchConfig(ctx context.Context) error {
	req := map[string]interface{}{
		"namespace": c.cfg.Namespace,
		"key":       c.cfg.ConfigKey,
		"client_id": c.cfg.ClientID,
	}

	url := fmt.Sprintf("%s/api/v1/config/get", c.cfg.ServerAddr)

	resp, err := c.client.Post(ctx, url, req)
	if err != nil {
		return fmt.Errorf("fetch config failed: %w", err)
	}
	defer resp.Close()

	var version common.ConfigVersion
	if err := json.Unmarshal(resp.ReadAll(), &version); err != nil {
		return fmt.Errorf("parse config response failed: %w", err)
	}

	g.Log().Infof(ctx, "config fetched: namespace=%s, key=%s, md5=%s",
		version.Namespace, version.Key, version.MD5)

	// 更新缓存
	c.cache.Set(&version)

	return nil
}

// notifyListeners 通知所有监听器
func (c *Client) notifyListeners(version *common.ConfigVersion) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	configKey := version.Namespace + "/" + version.Key
	listeners := c.listeners[configKey]

	for _, listener := range listeners {
		go listener(version)
	}
}

package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/krustd/gf-nexus/nexus-config/common"
)

// ChangeListener 配置变更监听器
type ChangeListener func(version *common.ConfigVersion)

// Client 配置中心客户端
type Client struct {
	cfg        *common.ClientConfig
	cache      *ConfigCache
	httpClient *http.Client // 通用请求（fetchConfig 等）
	pollClient *http.Client // 长轮询专用，无 ResponseHeaderTimeout 限制
	listeners  map[string][]ChangeListener
	mu         sync.RWMutex
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewClient 创建配置中心客户端
func NewClient(cfg *common.ClientConfig) *Client {
	pollTimeout := time.Duration(cfg.PollTimeout+10) * time.Second
	return &Client{
		cfg:   cfg,
		cache: NewConfigCache(),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		pollClient: &http.Client{
			Transport: &http.Transport{},
			Timeout:   pollTimeout,
		},
		listeners: make(map[string][]ChangeListener),
		stopCh:    make(chan struct{}),
	}
}

// Start 启动客户端（首次拉取 + 长轮询）
func (c *Client) Start(ctx context.Context) error {
	log.Printf("[nexus-config] starting, namespace=%s, key=%s", c.cfg.Namespace, c.cfg.ConfigKey)

	if err := c.fetchConfig(ctx); err != nil {
		log.Printf("[nexus-config] initial fetch failed: %v", err)
	}

	c.wg.Add(1)
	go c.longPollLoop(ctx)

	return nil
}

// Stop 停止客户端
func (c *Client) Stop() {
	close(c.stopCh)
	c.wg.Wait()
	log.Println("[nexus-config] client stopped")
}

// GetConfig 获取缓存中的配置
func (c *Client) GetConfig() (*common.ConfigVersion, error) {
	version, ok := c.cache.Get(c.cfg.Namespace, c.cfg.ConfigKey)
	if !ok {
		return nil, fmt.Errorf("config not found in cache")
	}
	return version, nil
}

// GetValue 获取配置内容字符串
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
	return common.ParseConfig(version.Value, common.ConfigFormat(version.Format), target)
}

// AddChangeListener 添加配置变更监听器
func (c *Client) AddChangeListener(listener ChangeListener) {
	c.mu.Lock()
	defer c.mu.Unlock()
	configKey := c.cfg.Namespace + "/" + c.cfg.ConfigKey
	c.listeners[configKey] = append(c.listeners[configKey], listener)
}

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
	currentMD5 := ""
	if version, ok := c.cache.Get(c.cfg.Namespace, c.cfg.ConfigKey); ok {
		currentMD5 = version.MD5
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"namespace": c.cfg.Namespace,
		"key":       c.cfg.ConfigKey,
		"client_id": c.cfg.ClientID,
		"md5":       currentMD5,
	})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.cfg.ServerAddr+"/api/v1/config/poll", bytes.NewReader(reqBody))
	if err != nil {
		log.Printf("[nexus-config] poll build request failed: %v", err)
		time.Sleep(time.Duration(c.cfg.RetryDelay) * time.Second)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.pollClient.Do(httpReq)
	if err != nil {
		log.Printf("[nexus-config] poll failed: %v", err)
		time.Sleep(time.Duration(c.cfg.RetryDelay) * time.Second)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[nexus-config] poll read body failed: %v", err)
		time.Sleep(time.Duration(c.cfg.RetryDelay) * time.Second)
		return
	}

	var pollResp struct {
		Changed bool                  `json:"changed"`
		Version *common.ConfigVersion `json:"version"`
	}
	if err := json.Unmarshal(body, &pollResp); err != nil {
		log.Printf("[nexus-config] poll parse response failed: %v", err)
		time.Sleep(time.Duration(c.cfg.RetryDelay) * time.Second)
		return
	}

	if pollResp.Changed && pollResp.Version != nil {
		log.Printf("[nexus-config] config changed: %s/%s md5=%s",
			pollResp.Version.Namespace, pollResp.Version.Key, pollResp.Version.MD5)
		c.cache.Set(pollResp.Version)
		c.notifyListeners(pollResp.Version)
	}
}

// fetchConfig 立即拉取配置（非长轮询）
func (c *Client) fetchConfig(ctx context.Context) error {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"namespace": c.cfg.Namespace,
		"key":       c.cfg.ConfigKey,
		"client_id": c.cfg.ClientID,
	})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.cfg.ServerAddr+"/api/v1/config/get", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("fetch config build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("fetch config failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("fetch config read body: %w", err)
	}

	var version common.ConfigVersion
	if err := json.Unmarshal(body, &version); err != nil {
		return fmt.Errorf("fetch config parse response: %w", err)
	}

	log.Printf("[nexus-config] config fetched: %s/%s md5=%s", version.Namespace, version.Key, version.MD5)
	c.cache.Set(&version)
	return nil
}

func (c *Client) notifyListeners(version *common.ConfigVersion) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	configKey := version.Namespace + "/" + version.Key
	for _, listener := range c.listeners[configKey] {
		go listener(version)
	}
}

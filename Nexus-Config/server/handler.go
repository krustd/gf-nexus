package server

import (
	"context"
	"hash/fnv"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/krustd/nexus-config/common"
	"github.com/krustd/nexus-config/storage"
)

type Handler struct {
	storage  storage.Storage
	notifier *ConfigNotifier
}

func NewHandler(storage storage.Storage, notifier *ConfigNotifier) *Handler {
	return &Handler{
		storage:  storage,
		notifier: notifier,
	}
}

// PollConfigReq 长轮询请求
type PollConfigReq struct {
	Namespace string `json:"namespace" v:"required"`
	Key       string `json:"key" v:"required"`
	ClientID  string `json:"client_id" v:"required"`
	MD5       string `json:"md5"` // 客户端当前配置的 MD5
}

// PollConfigResp 长轮询响应
type PollConfigResp struct {
	Changed bool                  `json:"changed"`
	Version *common.ConfigVersion `json:"version,omitempty"`
}

// PollConfig 长轮询配置
func (h *Handler) PollConfig(r *ghttp.Request) {
	var req PollConfigReq
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(&PollConfigResp{Changed: false})
		return
	}

	ctx := r.GetCtx()

	// 获取当前发布的配置
	item, err := h.storage.GetPublishedConfig(ctx, req.Namespace, req.Key)
	if err != nil {
		g.Log().Errorf(ctx, "get published config failed: %v", err)
		r.Response.WriteJson(&PollConfigResp{Changed: false})
		return
	}

	// 检查是否有灰度规则
	grayRule, err := h.storage.GetGrayRule(ctx, req.Namespace, req.Key)
	isGrayEnabled := err == nil && grayRule.Enabled

	// 计算当前应该使用的配置版本
	currentVersion := h.calculateVersion(ctx, item, req.ClientID, isGrayEnabled, grayRule)

	// 如果 MD5 不同，立即返回
	if currentVersion.MD5 != req.MD5 {
		g.Log().Infof(ctx, "config changed, client_md5=%s, server_md5=%s", req.MD5, currentVersion.MD5)
		r.Response.WriteJson(&PollConfigResp{
			Changed: true,
			Version: currentVersion,
		})
		return
	}

	// MD5 相同，等待配置变更（长轮询）
	g.Log().Infof(ctx, "config unchanged, waiting for change: %s/%s", req.Namespace, req.Key)

	// 等待 30 秒
	newVersion, changed := h.notifier.WaitForChange(ctx, req.Namespace, req.Key, 30*time.Second)

	if changed {
		// 重新计算灰度（因为可能灰度规则变了）
		item, err := h.storage.GetPublishedConfig(ctx, req.Namespace, req.Key)
		if err == nil {
			grayRule, err := h.storage.GetGrayRule(ctx, req.Namespace, req.Key)
			isGrayEnabled := err == nil && grayRule.Enabled
			newVersion = h.calculateVersion(ctx, item, req.ClientID, isGrayEnabled, grayRule)
		}

		r.Response.WriteJson(&PollConfigResp{
			Changed: true,
			Version: newVersion,
		})
		return
	}

	// 超时，返回未变更
	r.Response.WriteJson(&PollConfigResp{Changed: false})
}

// calculateVersion 计算当前客户端应该使用的配置版本（含灰度计算）
func (h *Handler) calculateVersion(ctx context.Context, item *common.ConfigItem, clientID string, isGrayEnabled bool, grayRule *common.GrayRule) *common.ConfigVersion {
	// 默认使用已发布版本
	value := item.PublishedValue
	md5str := item.PublishedMD5

	// 如果启用了灰度，并且客户端命中灰度
	if isGrayEnabled && h.hitGray(clientID, grayRule.Percentage) {
		g.Log().Infof(ctx, "client %s hit gray rule, percentage=%d", clientID, grayRule.Percentage)
		// 使用草稿版本
		if item.DraftValue != "" {
			value = item.DraftValue
			md5str = item.DraftMD5
		}
	}

	return &common.ConfigVersion{
		Namespace: item.Namespace,
		Key:       item.Key,
		MD5:       md5str,
		Value:     value,
		Format:    string(item.Format),
	}
}

// hitGray 判断是否命中灰度（基于 clientID 的哈希）
func (h *Handler) hitGray(clientID string, percentage int) bool {
	if percentage <= 0 {
		return false
	}
	if percentage >= 100 {
		return true
	}

	hash := fnv.New32a()
	hash.Write([]byte(clientID))
	hashValue := hash.Sum32()

	return int(hashValue%100) < percentage
}

// NotifyConfigChange 通知配置变更（供 Admin API 调用）
func (h *Handler) NotifyConfigChange(ctx context.Context, namespace, key string) error {
	item, err := h.storage.GetPublishedConfig(ctx, namespace, key)
	if err != nil {
		return err
	}

	version := &common.ConfigVersion{
		Namespace: item.Namespace,
		Key:       item.Key,
		MD5:       item.PublishedMD5,
		Value:     item.PublishedValue,
		Format:    string(item.Format),
	}

	h.notifier.Notify(ctx, version)
	return nil
}

// GetConfigReq 获取配置请求
type GetConfigReq struct {
	Namespace string `json:"namespace" v:"required"`
	Key       string `json:"key" v:"required"`
	ClientID  string `json:"client_id" v:"required"`
}

// GetConfig 获取配置（非长轮询，立即返回）
func (h *Handler) GetConfig(r *ghttp.Request) {
	var req GetConfigReq
	if err := r.Parse(&req); err != nil {
		r.Response.Status = 400
		r.Response.WriteJson(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	ctx := r.GetCtx()

	// 获取发布的配置
	item, err := h.storage.GetPublishedConfig(ctx, req.Namespace, req.Key)
	if err != nil {
		r.Response.Status = 404
		r.Response.WriteJson(map[string]interface{}{
			"error": "config not found",
		})
		return
	}

	// 检查灰度规则
	grayRule, err := h.storage.GetGrayRule(ctx, req.Namespace, req.Key)
	isGrayEnabled := err == nil && grayRule.Enabled

	version := h.calculateVersion(ctx, item, req.ClientID, isGrayEnabled, grayRule)

	r.Response.WriteJson(version)
}

// HealthCheck 健康检查
func (h *Handler) HealthCheck(r *ghttp.Request) {
	r.Response.WriteJson(map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Unix(),
	})
}

package admin

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/krustd/nexus-config/common"
	"github.com/krustd/nexus-config/storage"
)

// ConfigNotifier 配置变更通知接口
type ConfigNotifier interface {
	Notify(ctx context.Context, version *common.ConfigVersion)
}

type Handler struct {
	storage  storage.Storage
	notifier ConfigNotifier
}

func NewHandler(storage storage.Storage, notifier ConfigNotifier) *Handler {
	return &Handler{
		storage:  storage,
		notifier: notifier,
	}
}

// === Namespace 管理 ===

func (h *Handler) CreateNamespace(r *ghttp.Request) {
	var req CreateNamespaceReq
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(ErrorResp(400, err.Error()))
		return
	}

	ns := &common.ConfigNamespace{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.storage.CreateNamespace(context.Background(), ns); err != nil {
		r.Response.WriteJson(ErrorResp(500, err.Error()))
		return
	}

	r.Response.WriteJson(SuccessResp(ns))
}

func (h *Handler) ListNamespaces(r *ghttp.Request) {
	list, err := h.storage.ListNamespaces(context.Background())
	if err != nil {
		r.Response.WriteJson(ErrorResp(500, err.Error()))
		return
	}
	r.Response.WriteJson(SuccessResp(list))
}

func (h *Handler) GetNamespace(r *ghttp.Request) {
	id := r.Get("id").String()
	ns, err := h.storage.GetNamespace(context.Background(), id)
	if err != nil {
		r.Response.WriteJson(ErrorResp(404, "namespace not found"))
		return
	}
	r.Response.WriteJson(SuccessResp(ns))
}

func (h *Handler) DeleteNamespace(r *ghttp.Request) {
	id := r.Get("id").String()
	if err := h.storage.DeleteNamespace(context.Background(), id); err != nil {
		r.Response.WriteJson(ErrorResp(500, err.Error()))
		return
	}
	r.Response.WriteJson(SuccessResp(nil))
}

// === 配置管理 ===

func (h *Handler) SaveDraft(r *ghttp.Request) {
	var req SaveDraftReq
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(ErrorResp(400, err.Error()))
		return
	}

	if err := h.storage.SaveDraft(context.Background(), req.Namespace, req.Key, req.Value, req.Format); err != nil {
		r.Response.WriteJson(ErrorResp(500, err.Error()))
		return
	}

	r.Response.WriteJson(SuccessResp(nil))
}

func (h *Handler) GetDraft(r *ghttp.Request) {
	namespace := r.Get("namespace").String()
	key := r.Get("key").String()

	item, err := h.storage.GetDraft(context.Background(), namespace, key)
	if err != nil {
		r.Response.WriteJson(ErrorResp(404, "draft not found"))
		return
	}

	r.Response.WriteJson(SuccessResp(item))
}

func (h *Handler) PublishConfig(r *ghttp.Request) {
	var req PublishConfigReq
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(ErrorResp(400, err.Error()))
		return
	}

	ctx := context.Background()

	if err := h.storage.PublishConfig(ctx, req.Namespace, req.Key); err != nil {
		r.Response.WriteJson(ErrorResp(500, err.Error()))
		return
	}

	g.Log().Infof(ctx, "config published: %s/%s", req.Namespace, req.Key)

	// 通知配置变更
	if h.notifier != nil {
		// 获取发布后的配置并通知
		item, err := h.storage.GetPublishedConfig(ctx, req.Namespace, req.Key)
		if err == nil {
			version := &common.ConfigVersion{
				Namespace: item.Namespace,
				Key:       item.Key,
				MD5:       item.PublishedMD5,
				Value:     item.PublishedValue,
				Format:    string(item.Format),
			}
			h.notifier.Notify(ctx, version)
			g.Log().Infof(ctx, "config change notified: %s/%s", req.Namespace, req.Key)
		}
	}

	r.Response.WriteJson(SuccessResp(nil))
}

func (h *Handler) GetPublished(r *ghttp.Request) {
	namespace := r.Get("namespace").String()
	key := r.Get("key").String()

	item, err := h.storage.GetPublishedConfig(context.Background(), namespace, key)
	if err != nil {
		r.Response.WriteJson(ErrorResp(404, "published config not found"))
		return
	}

	r.Response.WriteJson(SuccessResp(item))
}

func (h *Handler) ListConfigs(r *ghttp.Request) {
	namespace := r.Get("namespace").String()

	list, err := h.storage.ListConfigs(context.Background(), namespace)
	if err != nil {
		r.Response.WriteJson(ErrorResp(500, err.Error()))
		return
	}

	r.Response.WriteJson(SuccessResp(list))
}

func (h *Handler) DeleteConfig(r *ghttp.Request) {
	namespace := r.Get("namespace").String()
	key := r.Get("key").String()

	if err := h.storage.DeleteConfig(context.Background(), namespace, key); err != nil {
		r.Response.WriteJson(ErrorResp(500, err.Error()))
		return
	}

	r.Response.WriteJson(SuccessResp(nil))
}

// === 灰度规则管理 ===

func (h *Handler) SaveGrayRule(r *ghttp.Request) {
	var req SaveGrayRuleReq
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(ErrorResp(400, err.Error()))
		return
	}

	rule := &common.GrayRule{
		Namespace:  req.Namespace,
		Key:        req.Key,
		Percentage: req.Percentage,
		Enabled:    req.Enabled,
	}

	if err := h.storage.SaveGrayRule(context.Background(), rule); err != nil {
		r.Response.WriteJson(ErrorResp(500, err.Error()))
		return
	}

	r.Response.WriteJson(SuccessResp(rule))
}

func (h *Handler) GetGrayRule(r *ghttp.Request) {
	namespace := r.Get("namespace").String()
	key := r.Get("key").String()

	rule, err := h.storage.GetGrayRule(context.Background(), namespace, key)
	if err != nil {
		r.Response.WriteJson(ErrorResp(404, "gray rule not found"))
		return
	}

	r.Response.WriteJson(SuccessResp(rule))
}

func (h *Handler) DeleteGrayRule(r *ghttp.Request) {
	namespace := r.Get("namespace").String()
	key := r.Get("key").String()

	if err := h.storage.DeleteGrayRule(context.Background(), namespace, key); err != nil {
		r.Response.WriteJson(ErrorResp(500, err.Error()))
		return
	}

	r.Response.WriteJson(SuccessResp(nil))
}

func (h *Handler) ListGrayRules(r *ghttp.Request) {
	namespace := r.Get("namespace").String()

	list, err := h.storage.ListGrayRules(context.Background(), namespace)
	if err != nil {
		r.Response.WriteJson(ErrorResp(500, err.Error()))
		return
	}

	r.Response.WriteJson(SuccessResp(list))
}

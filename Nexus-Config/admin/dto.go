package admin

import "github.com/krustd/gf-nexus/nexus-config/common"

// CreateNamespaceReq 创建命名空间请求
type CreateNamespaceReq struct {
	ID          string `json:"id" v:"required|length:1,64"`
	Name        string `json:"name" v:"required|length:1,128"`
	Description string `json:"description"`
}

// SaveDraftReq 保存草稿请求
type SaveDraftReq struct {
	Namespace string              `json:"namespace" v:"required"`
	Key       string              `json:"key" v:"required"`
	Value     string              `json:"value" v:"required"`
	Format    common.ConfigFormat `json:"format" v:"required|in:yaml,json,toml,properties"`
}

// PublishConfigReq 发布配置请求
type PublishConfigReq struct {
	Namespace string `json:"namespace" v:"required"`
	Key       string `json:"key" v:"required"`
}

// GetConfigReq 获取配置请求
type GetConfigReq struct {
	Namespace string `json:"namespace" v:"required"`
	Key       string `json:"key" v:"required"`
}

// DeleteConfigReq 删除配置请求
type DeleteConfigReq struct {
	Namespace string `json:"namespace" v:"required"`
	Key       string `json:"key" v:"required"`
}

// SaveGrayRuleReq 保存灰度规则请求
type SaveGrayRuleReq struct {
	Namespace  string `json:"namespace" v:"required"`
	Key        string `json:"key" v:"required"`
	Percentage int    `json:"percentage" v:"required|between:0,100"`
	Enabled    bool   `json:"enabled"`
}

// Response 通用响应
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// SuccessResp 成功响应
func SuccessResp(data interface{}) *Response {
	return &Response{
		Code: 0,
		Msg:  "success",
		Data: data,
	}
}

// ErrorResp 错误响应
func ErrorResp(code int, msg string) *Response {
	return &Response{
		Code: code,
		Msg:  msg,
	}
}

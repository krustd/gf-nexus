package gateway

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/krustd/nexus-gateway/config"
	"github.com/krustd/nexus-registry/registry"
)

// ProxyHandler 泛化调用反向代理
type ProxyHandler struct {
	pool       *ResolverPool
	httpClient *http.Client
	grpcProxy  *GRPCProxy
}

func NewProxyHandler(pool *ResolverPool, cfg config.TimeoutConfig, grpcProxy *GRPCProxy) *ProxyHandler {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: time.Duration(cfg.ConnectMs) * time.Millisecond,
		}).DialContext,
		MaxIdleConnsPerHost: 100,
		MaxIdleConns:        500,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(cfg.ResponseMs) * time.Millisecond,
	}
	return &ProxyHandler{
		pool:       pool,
		httpClient: client,
		grpcProxy:  grpcProxy,
	}
}

// Handle 处理 /api/:service/*method 的泛化调用
func (p *ProxyHandler) Handle(r *ghttp.Request) {
	ctx := r.GetCtx()
	serviceName := r.GetRouter("service").String()
	method := r.GetRouter("method").String()

	// 校验服务名
	if serviceName == "" {
		GatewayError(r, CodeServiceNotFound, "empty service name")
		return
	}

	// 去除 method 前导斜杠
	method = strings.TrimPrefix(method, "/")

	// 1. 服务发现 + 负载均衡
	resolver, err := p.pool.GetOrCreate(serviceName)
	if err != nil {
		g.Log().Errorf(ctx, "[gateway] resolver create failed: %s: %v", serviceName, err)
		GatewayError(r, CodeServiceNotFound, fmt.Sprintf("service not found: %s", serviceName))
		return
	}

	instance, err := resolver.Resolve()
	if err != nil {
		g.Log().Errorf(ctx, "[gateway] resolve failed: %s: %v", serviceName, err)
		GatewayError(r, CodeServiceNotFound, fmt.Sprintf("no available instance for %s", serviceName))
		return
	}

	// 按协议分发
	if instance.Protocol == registry.ProtocolGRPC {
		p.grpcProxy.Handle(r, instance.Address, method)
		return
	}

	// 2. 构建目标 URL（HTTP）
	targetURL := fmt.Sprintf("http://%s/%s", instance.Address, method)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// 3. 创建转发请求
	proxyReq, err := http.NewRequestWithContext(ctx, r.Method, targetURL, r.Body)
	if err != nil {
		g.Log().Errorf(ctx, "[gateway] create proxy request failed: %v", err)
		GatewayError(r, CodeBackendError, "failed to create proxy request")
		return
	}

	// 4. 拷贝请求头（排除 hop-by-hop headers）
	copyRequestHeaders(r.Request.Header, proxyReq.Header)

	// 5. 执行转发
	resp, err := p.httpClient.Do(proxyReq)
	if err != nil {
		g.Log().Errorf(ctx, "[gateway] proxy to %s failed: %v", targetURL, err)
		if isTimeout(err) {
			GatewayError(r, CodeBackendTimeout, fmt.Sprintf("backend timeout: %s", serviceName))
		} else {
			GatewayError(r, CodeBackendError, fmt.Sprintf("backend error: %s", serviceName))
		}
		return
	}
	defer resp.Body.Close()

	// 6. 拷贝响应头
	copyResponseHeaders(resp.Header, r.Response.Header())

	// 7. 写入状态码和 body
	r.Response.WriteStatus(resp.StatusCode)
	if _, err := io.Copy(r.Response.RawWriter(), resp.Body); err != nil {
		g.Log().Errorf(ctx, "[gateway] copy response body failed: %v", err)
	}
}

// hop-by-hop headers 不应被转发
var hopHeaders = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"Te":                  true,
	"Trailer":             true,
	"Transfer-Encoding":   true,
	"Upgrade":             true,
}

func copyRequestHeaders(src, dst http.Header) {
	for k, vv := range src {
		if hopHeaders[k] {
			continue
		}
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func copyResponseHeaders(src, dst http.Header) {
	for k, vv := range src {
		if hopHeaders[k] {
			continue
		}
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func isTimeout(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}

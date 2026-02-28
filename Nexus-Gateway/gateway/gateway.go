package gateway

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/krustd/gf-nexus/nexus-gateway/config"
	"github.com/krustd/gf-nexus/nexus-gateway/metrics"
	"github.com/krustd/gf-nexus/nexus-gateway/middleware"
	"github.com/krustd/gf-nexus/nexus-registry/registry"
)

// Gateway API 网关核心
type Gateway struct {
	config    *config.GatewayConfig
	holder    *config.DynamicConfigHolder
	pool      *ResolverPool
	reg       registry.Registry
	server    *ghttp.Server
	grpcProxy *GRPCProxy
	keyMgr    *middleware.KeyManager
}

func New(cfg *config.GatewayConfig, holder *config.DynamicConfigHolder, reg registry.Registry) (*Gateway, error) {
	dynCfg := holder.Load()
	pickerFactory := newPickerFactory(dynCfg.Balancer.Strategy)
	pool := NewResolverPool(reg, pickerFactory)

	// 创建 JWT 密钥管理器
	km := middleware.NewKeyManager()
	km.UpdateKeys(dynCfg.JWT.Keys)

	gw := &Gateway{
		config: cfg,
		holder: holder,
		pool:   pool,
		reg:    reg,
		keyMgr: km,
	}

	// 注册动态配置变更回调
	holder.OnChange(func(newCfg *config.DynamicConfig) {
		// 更新 JWT 密钥
		km.UpdateKeys(newCfg.JWT.Keys)

		// 更新负载均衡策略
		if newCfg.Balancer.Strategy != "" {
			pool.UpdateStrategy(newCfg.Balancer.Strategy)
		}
	})

	return gw, nil
}

// Start 启动网关（阻塞）
func (gw *Gateway) Start() {
	s := g.Server("gateway")
	s.SetAddr(gw.config.Server.Addr)
	gw.server = s

	// 全局中间件链（顺序重要）
	s.Use(
		middleware.Trace(),
		middleware.RequestID(),
		middleware.Logging(),
		middleware.CORS(gw.holder),
		middleware.IPFilter(gw.holder),
		middleware.RateLimit(gw.holder),
		middleware.JWT(gw.holder, gw.keyMgr),
	)

	// 健康检查
	s.BindHandler("GET:/health", func(r *ghttp.Request) {
		r.Response.WriteJson(g.Map{
			"status": "ok",
			"time":   time.Now().Unix(),
		})
	})

	// Prometheus 指标
	if gw.config.Metrics.Enabled {
		metrics.Register(s, gw.config.Metrics.Path)
	}

	// 泛化调用路由
	gw.grpcProxy = NewGRPCProxy(gw.config.GRPC)
	proxy := NewProxyHandler(gw.pool, gw.config.Timeout, gw.grpcProxy)
	cb := middleware.NewCircuitBreakerManager(gw.holder)

	s.BindHandler("ALL:/api/:service/*method", func(r *ghttp.Request) {
		serviceName := r.GetRouter("service").String()

		// 熔断检查
		if cb.Enabled() && !cb.Allow(serviceName) {
			GatewayError(r, CodeCircuitOpen, "circuit breaker open for "+serviceName)
			return
		}

		// 执行代理
		proxy.Handle(r)

		// 记录熔断指标
		if cb.Enabled() {
			status := r.Response.Status
			if status >= 500 {
				cb.RecordFailure(serviceName)
			} else {
				cb.RecordSuccess(serviceName)
			}
		}
	})

	s.Run()
}

// Shutdown 优雅关闭
func (gw *Gateway) Shutdown() {
	if gw.grpcProxy != nil {
		gw.grpcProxy.Close()
	}
	gw.pool.Close()
	if gw.reg != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		gw.reg.Close(ctx)
	}
}

package gateway

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/krustd/nexus-gateway/config"
	"github.com/krustd/nexus-gateway/metrics"
	"github.com/krustd/nexus-gateway/middleware"
	"github.com/krustd/nexus-registry/registry"
	"github.com/krustd/nexus-registry/registry/balancer"
)

// Gateway API 网关核心
type Gateway struct {
	config *config.GatewayConfig
	pool   *ResolverPool
	reg    registry.Registry
	server *ghttp.Server
}

func New(cfg *config.GatewayConfig, reg registry.Registry) (*Gateway, error) {
	pickerFactory := newPickerFactory(cfg.Balancer.Strategy)
	pool := NewResolverPool(reg, pickerFactory)

	return &Gateway{
		config: cfg,
		pool:   pool,
		reg:    reg,
	}, nil
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
		middleware.CORS(gw.config.CORS),
		middleware.IPFilter(gw.config.IPFilter),
		middleware.RateLimit(gw.config.RateLimit),
		middleware.JWT(gw.config.JWT),
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
	proxy := NewProxyHandler(gw.pool, gw.config.Timeout)
	cb := middleware.NewCircuitBreakerManager(gw.config.Circuit)

	s.BindHandler("ALL:/api/:service/*method", func(r *ghttp.Request) {
		serviceName := r.GetRouter("service").String()

		// 熔断检查
		if gw.config.Circuit.Enabled && !cb.Allow(serviceName) {
			GatewayError(r, CodeCircuitOpen, "circuit breaker open for "+serviceName)
			return
		}

		// 执行代理
		proxy.Handle(r)

		// 记录熔断指标
		if gw.config.Circuit.Enabled {
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
	gw.pool.Close()
	if gw.reg != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		gw.reg.Close(ctx)
	}
}

func newPickerFactory(strategy string) func() registry.Picker {
	return func() registry.Picker {
		switch strategy {
		case "random":
			return balancer.NewRandom()
		case "weighted_round_robin":
			return balancer.NewWeightedRoundRobin()
		default:
			return balancer.NewRoundRobin()
		}
	}
}

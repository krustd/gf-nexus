package metrics

import (
	"strconv"
	"strings"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	RequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_requests_total",
			Help: "Total number of requests processed by the gateway",
		},
		[]string{"method", "service", "status"},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_request_duration_seconds",
			Help:    "Request latency distribution",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "service"},
	)

	CircuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_circuit_breaker_state",
			Help: "Circuit breaker state per service (0=closed, 1=open, 2=half-open)",
		},
		[]string{"service"},
	)
)

func init() {
	prometheus.MustRegister(RequestTotal, RequestDuration, CircuitBreakerState)
}

// Register 绑定 Prometheus metrics 路由
func Register(s *ghttp.Server, path string) {
	s.BindHandler("GET:"+path, func(r *ghttp.Request) {
		promhttp.Handler().ServeHTTP(r.Response.RawWriter(), r.Request)
	})
}

// RecordRequest 记录一次请求的指标
func RecordRequest(method, path string, status int, latency time.Duration) {
	service := extractService(path)
	statusStr := strconv.Itoa(status)

	RequestTotal.WithLabelValues(method, service, statusStr).Inc()
	RequestDuration.WithLabelValues(method, service).Observe(latency.Seconds())
}

// extractService 从路径 /api/:service/... 提取 service 名
func extractService(path string) string {
	// /api/user-service/v1/users → user-service
	parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 3)
	if len(parts) >= 2 && parts[0] == "api" {
		return parts[1]
	}
	return "unknown"
}

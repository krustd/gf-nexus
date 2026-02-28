package gateway

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/krustd/gf-nexus/nexus-gateway/config"
)

// GRPCProxy 处理 HTTP→gRPC 转码（仅 Unary）
type GRPCProxy struct {
	cfg config.GRPCConfig

	connMu sync.RWMutex
	conns  map[string]*grpc.ClientConn

	cacheMu sync.RWMutex
	cache   map[string]*cachedDescriptor
}

type cachedDescriptor struct {
	svcDesc  *desc.ServiceDescriptor
	cachedAt time.Time
}

func NewGRPCProxy(cfg config.GRPCConfig) *GRPCProxy {
	return &GRPCProxy{
		cfg:   cfg,
		conns: make(map[string]*grpc.ClientConn),
		cache: make(map[string]*cachedDescriptor),
	}
}

func (gp *GRPCProxy) getOrCreateConn(ctx context.Context, address string) (*grpc.ClientConn, error) {
	gp.connMu.RLock()
	conn, ok := gp.conns[address]
	gp.connMu.RUnlock()
	if ok {
		return conn, nil
	}

	gp.connMu.Lock()
	defer gp.connMu.Unlock()

	if conn, ok := gp.conns[address]; ok {
		return conn, nil
	}

	dialCtx, cancel := context.WithTimeout(ctx, time.Duration(gp.cfg.ConnectTimeoutMs)*time.Millisecond)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc dial %s: %w", address, err)
	}
	gp.conns[address] = conn
	return conn, nil
}

func (gp *GRPCProxy) resolveMethod(ctx context.Context, conn *grpc.ClientConn, address, fullMethod string) (*desc.MethodDescriptor, error) {
	slashIdx := strings.LastIndex(fullMethod, "/")
	if slashIdx < 0 {
		return nil, fmt.Errorf("invalid gRPC method path: %s (expected Service/Method)", fullMethod)
	}
	fullServiceName := fullMethod[:slashIdx]
	methodName := fullMethod[slashIdx+1:]

	cacheKey := address + "|" + fullServiceName
	gp.cacheMu.RLock()
	cached, ok := gp.cache[cacheKey]
	gp.cacheMu.RUnlock()

	ttl := time.Duration(gp.cfg.ReflectionCacheTTLSec) * time.Second
	if ok && time.Since(cached.cachedAt) < ttl {
		md := cached.svcDesc.FindMethodByName(methodName)
		if md == nil {
			return nil, fmt.Errorf("method %s not found in service %s", methodName, fullServiceName)
		}
		return md, nil
	}

	refClient := grpcreflect.NewClientAuto(ctx, conn)
	defer refClient.Reset()

	svcDesc, err := refClient.ResolveService(fullServiceName)
	if err != nil {
		return nil, fmt.Errorf("reflection resolve %s: %w", fullServiceName, err)
	}

	gp.cacheMu.Lock()
	gp.cache[cacheKey] = &cachedDescriptor{svcDesc: svcDesc, cachedAt: time.Now()}
	gp.cacheMu.Unlock()

	md := svcDesc.FindMethodByName(methodName)
	if md == nil {
		return nil, fmt.Errorf("method %s not found in service %s", methodName, fullServiceName)
	}
	return md, nil
}

// Handle 处理 HTTP→gRPC 转码
func (gp *GRPCProxy) Handle(r *ghttp.Request, address, method string) {
	ctx := r.GetCtx()

	// 1. 获取/创建 gRPC 连接
	conn, err := gp.getOrCreateConn(ctx, address)
	if err != nil {
		g.Log().Errorf(ctx, "[gateway] grpc connect failed: %s: %v", address, err)
		GatewayError(r, CodeBackendError, fmt.Sprintf("grpc connect failed: %s", address))
		return
	}

	// 2. 通过反射解析方法描述符
	md, err := gp.resolveMethod(ctx, conn, address, method)
	if err != nil {
		g.Log().Errorf(ctx, "[gateway] grpc resolve method failed: %s: %v", method, err)
		GatewayError(r, CodeBackendError, fmt.Sprintf("grpc method not found: %s", method))
		return
	}

	// 3. 拒绝 streaming 方法
	if md.IsClientStreaming() || md.IsServerStreaming() {
		GatewayError(r, CodeBackendError, fmt.Sprintf("streaming not supported: %s", method))
		return
	}

	// 4. 读取请求体，JSON → protobuf
	body, err := io.ReadAll(r.Body)
	if err != nil {
		GatewayError(r, CodeBackendError, "failed to read request body")
		return
	}

	reqMsg := dynamic.NewMessage(md.GetInputType())
	if len(body) > 0 {
		if err := reqMsg.UnmarshalJSON(body); err != nil {
			GatewayError(r, CodeBackendError, fmt.Sprintf("invalid JSON for %s: %v", method, err))
			return
		}
	}

	// 5. 转发关键 HTTP 头为 gRPC metadata
	outMD := metadata.MD{}
	for _, key := range []string{"Authorization", "X-Request-Id", "X-Trace-Id", "X-User-Id", "X-User-Role"} {
		if val := r.Header.Get(key); val != "" {
			outMD.Set(strings.ToLower(key), val)
		}
	}
	callCtx := metadata.NewOutgoingContext(ctx, outMD)

	// 6. 设置调用超时
	callCtx, cancel := context.WithTimeout(callCtx, time.Duration(gp.cfg.RequestTimeoutMs)*time.Millisecond)
	defer cancel()

	// 7. 调用 Unary RPC
	stub := grpcdynamic.NewStub(conn)
	respMsg, err := stub.InvokeRpc(callCtx, md, reqMsg)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			httpStatus := grpcCodeToHTTP(st.Code())
			r.Response.WriteStatus(httpStatus)
			r.Response.WriteJsonExit(g.Map{
				"code":    int(st.Code()),
				"message": st.Message(),
			})
		} else {
			GatewayError(r, CodeBackendError, fmt.Sprintf("grpc call failed: %v", err))
		}
		return
	}

	// 8. protobuf → JSON 响应
	respJSON, err := respMsg.(*dynamic.Message).MarshalJSON()
	if err != nil {
		GatewayError(r, CodeBackendError, fmt.Sprintf("marshal response failed: %v", err))
		return
	}

	r.Response.Header().Set("Content-Type", "application/json")
	r.Response.WriteStatus(200)
	r.Response.Write(respJSON)
}

// grpcCodeToHTTP 将 gRPC 状态码映射为 HTTP 状态码
func grpcCodeToHTTP(code codes.Code) int {
	switch code {
	case codes.OK:
		return 200
	case codes.InvalidArgument:
		return 400
	case codes.Unauthenticated:
		return 401
	case codes.PermissionDenied:
		return 403
	case codes.NotFound:
		return 404
	case codes.AlreadyExists:
		return 409
	case codes.ResourceExhausted:
		return 429
	case codes.Unimplemented:
		return 501
	case codes.Unavailable:
		return 503
	case codes.DeadlineExceeded:
		return 504
	default:
		return 500
	}
}

// Close 关闭所有 gRPC 连接
func (gp *GRPCProxy) Close() {
	gp.connMu.Lock()
	defer gp.connMu.Unlock()
	for addr, conn := range gp.conns {
		conn.Close()
		delete(gp.conns, addr)
	}
}

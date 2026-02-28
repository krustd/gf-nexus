package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	nexus "github.com/krustd/gf-nexus/nexus-registry"

	pb "github.com/krustd/gf-nexus/hello-service/proto"
)

// helloServer 实现 HelloService
type helloServer struct {
	pb.UnimplementedHelloServiceServer
}

func (s *helloServer) SayHello(_ context.Context, _ *pb.HelloRequest) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{
		Message: "Hello, World!",
		Service: "hello-service",
	}, nil
}

func (s *helloServer) SayHelloByName(_ context.Context, req *pb.NameRequest) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{
		Message: "Hello, " + req.Name + "!",
		Service: "hello-service",
	}, nil
}

func main() {
	// 注册到 etcd
	nexus.MustSetup("config/config.toml")
	defer nexus.Shutdown()

	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterHelloServiceServer(grpcServer, &helloServer{})

	// 开启 gRPC 反射（网关通过反射做 JSON↔Protobuf 转码）
	reflection.Register(grpcServer)

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("[hello-service] shutting down...")
		grpcServer.GracefulStop()
	}()

	log.Println("[hello-service] gRPC server listening on :9090")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

package main

import (
    "flag"
    "fmt"
    "log"
    "net"

    "grpcdemo/consul"
    "grpcdemo/greeter_server/services"
    pb "grpcdemo/pb/helloworld"
    "grpcdemo/utils"

    "google.golang.org/grpc"
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
    "google.golang.org/grpc/reflection"
)

var port = flag.Int("port", 12123, "The server port")

func main() {

    flag.Parse()

    lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))

    if err != nil {
        log.Fatalf("failed to listen :%v", err)
    }

    s := grpc.NewServer()

    // gRPC服务是使用Protobuf(PB)协议的，而PB提供了在运行时获取Proto定义信息的反射功能。
    // grpc-go中的"google.golang.org/grpc/reflection"包就对这个反射功能提供了支持。
    // 通过该反射我们就可以使用类似 grpcurl 的终端工具测试 rpc 接口了
    reflection.Register(s)

    // 健康检查
    // 官方文档：https://github.com/grpc/grpc/blob/master/doc/health-checking.md
    // gRPC-go 提供了健康检测库：https://pkg.go.dev/google.golang.org/grpc/health?tab=doc 把上面的文档接口化了。
    grpc_health_v1.RegisterHealthServer(s, health.NewServer())

    // 注册service
    //pb.RegisterGreeterServer(s, new(services.HelloService))
    pb.RegisterGreeterServer(s, &services.HelloService{})

    log.Printf("server listen at %v", lis.Addr())
    cc, err := consul.NewConsul("127.0.0.1:8500") // TODO Consul Cluster IP Config

    if err != nil {
        log.Fatalf("failed to conn consul:%v", err)
    }

    locIP, err := utils.GetClientIp()
    if err != nil {
        log.Fatalf("failed to get local ip:%v", err)
    }
    err = cc.RegisterService(&consul.ServiceEndpoint{
        IP:   locIP,
        Port: *port,
        Tag:  []string{"debug"},
        Name: "hello.service",
    })

    if err != nil {
        log.Fatalf("failed to RegisterService:%v", err)
    }

    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve:%v", err)
    }
}

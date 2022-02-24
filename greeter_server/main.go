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

const (
    consulAddr = "localhost:8500" // TODO Consul Cluster IP Config
)

func main() {

    flag.Parse()

    lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))

    if err != nil {
        log.Fatalf("failed to listen :%v", err)
    }

    s := grpc.NewServer()

    reflection.Register(s)
    grpc_health_v1.RegisterHealthServer(s, health.NewServer())

    // 注册service
    //pb.RegisterGreeterServer(s, new(services.HelloService))
    pb.RegisterGreeterServer(s, &services.HelloService{})

    log.Printf("server listen at %v", lis.Addr())
    cc, err := consul.NewConsul(consulAddr)

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

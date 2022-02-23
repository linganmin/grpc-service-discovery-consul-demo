package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    "grpcdemo/lb"
    pb "grpcdemo/pb/helloworld"

    "google.golang.org/grpc"
    "google.golang.org/grpc/balancer/roundrobin"
    "google.golang.org/grpc/credentials/insecure"
)

const defaultName = "world"

var (
    name = flag.String("name", defaultName, "name to reply")
)

func main() {

    flag.Parse()
    lb.Init()

    target := "consul://localhost:8500/hello.service" // TODO consul cluster

    conn, err := grpc.Dial(target,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingConfig": [{"%s":{}}]}`, roundrobin.Name)))

    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }

    defer conn.Close()

    c := pb.NewGreeterClient(conn)

    ctx, cancel := context.WithTimeout(context.Background(), time.Second)

    defer cancel()

    r, err := c.SayHello(ctx, &pb.HelloReq{Name: *name})

    if err != nil {
        log.Fatalf("could not greet:%v ", err)
    }

    log.Printf("Greeting: %s", r.Message)
}

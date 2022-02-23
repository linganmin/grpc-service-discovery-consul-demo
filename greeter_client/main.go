package main

import (
    "context"
    "flag"
    "log"
    "time"

    "grpcdemo/lb"
    pb "grpcdemo/pb/helloworld"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

const defaultName = "world"

var (
    name = flag.String("name", defaultName, "name to reply")
)

func main() {

    flag.Parse()
    lb.Init()

    //conn, err := grpc.Dial(*addr, grpc.WithInsecure())
    // TODO consul cluster
    //conn, err := grpc.Dial("consul://localhost:8500/hello.service?tag=dev&a=b&c=d")
    conn, err := grpc.Dial("consul://localhost:8500/hello.service", grpc.WithTransportCredentials(insecure.NewCredentials()))

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

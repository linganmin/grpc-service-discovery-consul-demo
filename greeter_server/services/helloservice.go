package services

import (
    "context"

    pb "grpcdemo/pb/helloworld"
)

type HelloService struct {
    pb.UnimplementedGreeterServer
}

func (h *HelloService) SayHello(ctx context.Context, req *pb.HelloReq) (*pb.HelloResp, error) {

    return &pb.HelloResp{Message: req.Name}, nil
}

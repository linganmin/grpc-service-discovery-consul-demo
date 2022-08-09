package services

import (
	"context"
	"time"

	pb "grpcdemo/pb/helloworld"
)

type HelloService struct {
	pb.UnimplementedGreeterServer
}

func (h *HelloService) SayHello(ctx context.Context, req *pb.HelloReq) (*pb.HelloResp, error) {

	time.Sleep(time.Second * 1) // 模拟耗时操作
	return &pb.HelloResp{Message: req.Name}, nil
}

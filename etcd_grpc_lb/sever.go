package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"etcd_grpc_lb/balancer"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "etcd_grpc_lb/helloworld"
)

const svcName = "hello_service"

var addr = "127.0.0.1:50054"

func main() {
	flag.StringVar(&addr, "addr", addr, "addr to lis")
	flag.Parse()

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %s", err)
	}
	defer lis.Close()

	s := grpc.NewServer()
	defer s.GracefulStop()
	pb.RegisterGreeterServer(s, &server{})

	//服务注册到etcd
	go  balancer.Register("127.0.0.1:2379", svcName,  addr, 5)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM)
	go func() {
		s := <-ch
		balancer.UnRegister(svcName, addr)

		if i, ok := s.(syscall.Signal); ok {
			os.Exit(int(i))
		} else {
			os.Exit(0)
		}

	}()

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message:"chen xun"},nil
}



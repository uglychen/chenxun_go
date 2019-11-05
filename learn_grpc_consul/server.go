package main

import (
    "fmt"
    "net"
    "google.golang.org/grpc/reflection"
    "golang.org/x/net/context"
    "google.golang.org/grpc"
    "flag"
    "time"

    register "learn_grpc_consul/registerConsul"
    pb "learn_grpc_consul/helloWorld"
)



var (
    serviceName = flag.String("serviceName", "HelloWorldService", "the name used register consul")
    host = flag.String("host", "192.168.1.8", "ip address")
    port = flag.Int("port", 9000, "the service listen port")
    consul_port = flag.Int("consul_port", 8500, "connect consual http port")
)

type HelloWordInterface struct {
}

func (s *HelloWordInterface) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
    fmt.Println("client called! 8081")
    return &pb.HelloReply{Message: "hi," + in.Name + "!"}, nil
}


func main(){

    fmt.Println("!!=====start listen")
    fmt.Println(*serviceName)
    fmt.Println(*host)
    fmt.Println(*port)
    fmt.Println(*consul_port)
    //listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *host, *port))
    listen, err := net.ListenTCP("tcp", &net.TCPAddr{net.ParseIP(*host), *port, ""})

    if err != nil {
        fmt.Println("chenxun" + err.Error())
    }

    s := grpc.NewServer()

    // start register service
    fmt.Println("!!=====start register the "+ *serviceName + " into sonsul ===")

    registerConsul := register.NewConsulRegister(fmt.Sprintf("%s:%d", *host, *consul_port), 15)
    registerBody := &register.RegisterInfo{
        Host:           *host,
        Port:           *port,
        ServiceName:    *serviceName,
        UpdateInterval: time.Second,
    }

    registerConsul.Register(*registerBody)

    pb.RegisterHelloServerServer(s, &HelloWordInterface{})
    reflection.Register(s)
    if err := s.Serve(listen); err != nil {
        fmt.Println("failed to serve:" + err.Error())
    }
}
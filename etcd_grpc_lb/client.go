package main

import (
	"flag"
	"fmt"
	"strconv"
	"time"

	"etcd_grpc_lb/balancer"
	pb "etcd_grpc_lb/helloworld"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

var (
	ServiceName = flag.String("ServiceName", "hello_service", "service name")
	EtcdAddr  = flag.String("EtcdAddr", "127.0.0.1:2379", "register etcd address")
)

func main() {
	flag.Parse()
	r := balancer.NewResolver(*EtcdAddr)
	resolver.Register(r)
	// "://author/" 为啥加这个还没有搞明白
	conn, err := grpc.Dial( r.Scheme()+"://author/"+ *ServiceName, grpc.WithBalancerName("round_robin"), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	ticker := time.NewTicker(2 * time.Second)
	for t := range ticker.C {
		client := pb.NewGreeterClient(conn)
		resp, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "world " + strconv.Itoa(t.Second())})
		if err == nil {
			fmt.Printf("%v: Reply is %s\n", t, resp.Message)
		}else{
			fmt.Printf("call server error:%s\n", err)
		}
	}
}
